package provider

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/dokku/docket/sdk"
	"github.com/dokku/docket/sdk/dokkutest"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestDomainsAreASet(t *testing.T) {
	server := dokkutest.New(t).SeedApp("api")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeProviders(server),
		Steps: []resource.TestStep{
			{
				Config: `resource "dokku_domains" "api" {
  app     = "api"
  domains = ["b.example.com", "a.example.com"]
}`,
				Check: checkServer(func() error {
					got := server.Domains("api")
					slices.Sort(got)
					if want := []string{"a.example.com", "b.example.com"}; !slices.Equal(got, want) {
						return fmt.Errorf("domains = %v, want %v", got, want)
					}
					return nil
				}),
			},
			{
				ResourceName:      "dokku_domains.api",
				ImportState:       true,
				ImportStateId:     "dokku_domains[app=api]",
				ImportStateVerify: true,
			},
		},
		CheckDestroy: func(*terraform.State) error {
			if got := server.Domains("api"); len(got) != 0 {
				return fmt.Errorf("domains = %v after destroy", got)
			}
			return nil
		},
	})
}

func TestPortsAreASetOfObjects(t *testing.T) {
	server := dokkutest.New(t).SeedApp("api")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeProviders(server),
		Steps: []resource.TestStep{
			{
				Config: `resource "dokku_ports" "api" {
  app = "api"
  port_mappings = [
    { scheme = "http", host = 80, container = 5000 },
    { scheme = "https", host = 443, container = 5000 },
  ]
}`,
				Check: checkServer(func() error {
					if got := server.Ports("api"); len(got) != 2 {
						return fmt.Errorf("ports = %v, want two", got)
					}
					return nil
				}),
			},
			{
				ResourceName:      "dokku_ports.api",
				ImportState:       true,
				ImportStateId:     "dokku_ports[app=api]",
				ImportStateVerify: true,
			},
		},
	})
}

func TestEmptyCollectionStaysInSync(t *testing.T) {
	server := dokkutest.New(t).SeedApp("api")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeProviders(server),
		Steps: []resource.TestStep{
			// Left out and written empty both declare an empty collection,
			// and neither plans a change once applied.
			{Config: `resource "dokku_domains" "api" { app = "api" }`},
			{Config: `resource "dokku_domains" "api" {
  app     = "api"
  domains = []
}`},
		},
	})
}

func TestServiceAndLink(t *testing.T) {
	server := dokkutest.New(t).SeedDatastore("redis", "redis:7.2").SeedApp("api")
	config := `
resource "dokku_service_create" "cache" {
  service = "redis"
  name    = "cache"
}

resource "dokku_service_link" "api" {
  app     = "api"
  service = dokku_service_create.cache.service
  name    = dokku_service_create.cache.name
}`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeProviders(server),
		Steps: []resource.TestStep{
			{
				// The image the service was created on is read back without
				// a diff, though the configuration pins none.
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("dokku_service_create.cache", "image"),
					checkServer(func() error {
						if got := server.Links("redis", "cache"); !slices.Equal(got, []string{"api"}) {
							return fmt.Errorf("links = %v", got)
						}
						return nil
					}),
				),
			},
			{
				// Destroying the app out of band drops the link, whose app
				// is gone, and plans it again.
				PreConfig: func() {
					outOfBand(t, server, "dokku_app", map[string]any{"app": "api", "state": "absent"})
				},
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestMissingAppRemovesDependents(t *testing.T) {
	server := dokkutest.New(t)
	config := `
resource "dokku_app" "api" { app = "api" }

resource "dokku_config" "api" {
  app     = dokku_app.api.app
  restart = false
  config  = { KEY = "value" }
}`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeProviders(server),
		Steps: []resource.TestStep{
			{Config: config},
			{
				PreConfig: func() {
					outOfBand(t, server, "dokku_app", map[string]any{"app": "api", "state": "absent"})
				},
				Config: config,
				Check: checkServer(func() error {
					if got := server.Config("api")["KEY"]; got != "value" {
						return fmt.Errorf("KEY = %q after the app was recreated", got)
					}
					return nil
				}),
			},
		},
	})
}

func TestPropertyTask(t *testing.T) {
	server := dokkutest.New(t).SeedApp("api")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeProviders(server),
		Steps: []resource.TestStep{
			{
				Config: `resource "dokku_nginx_property" "api" {
  app      = "api"
  property = "client-max-body-size"
  value    = "10m"
}`,
				Check: checkServer(func() error {
					if got, _ := server.Property("nginx", "api", "client-max-body-size"); got != "10m" {
						return fmt.Errorf("client-max-body-size = %q", got)
					}
					return nil
				}),
			},
		},
		CheckDestroy: func(*terraform.State) error {
			if got, ok := server.Property("nginx", "api", "client-max-body-size"); ok && got != "" {
				return fmt.Errorf("client-max-body-size = %q after destroy", got)
			}
			return nil
		},
	})
}

func TestVersionFloorRefusesApply(t *testing.T) {
	server := dokkutest.New(t).SeedVersion("0.1.0")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeProviders(server),
		Steps: []resource.TestStep{
			{
				Config:      `resource "dokku_app" "api" { app = "api" }`,
				ExpectError: regexp.MustCompile(`or later is required`),
			},
		},
	})
}

func TestVersionIsReadOnce(t *testing.T) {
	server := dokkutest.New(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeProviders(server),
		Steps: []resource.TestStep{
			{
				Config: `
resource "dokku_app" "a" { app = "a" }
resource "dokku_app" "b" { app = "b" }
resource "dokku_app" "c" { app = "c" }`,
				Check: checkServer(func() error {
					// The apply creates three apps under one session, which
					// reads the server's version once.
					if n := server.Count("version"); n != 1 {
						return fmt.Errorf("dokku version was read %d times", n)
					}
					return nil
				}),
			},
		},
	})
}

func TestParallelOperationsOnOneApp(t *testing.T) {
	server := dokkutest.New(t).SeedApp("api")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeProviders(server),
		Steps: []resource.TestStep{
			{
				Config: `
resource "dokku_config" "api" {
  app     = "api"
  restart = false
  config  = { KEY = "value" }
}

resource "dokku_domains" "api" {
  app     = "api"
  domains = ["api.example.com"]
}

resource "dokku_buildpacks" "api" {
  app        = "api"
  buildpacks = ["https://github.com/heroku/heroku-buildpack-go.git"]
}`,
				Check: checkServer(func() error {
					if len(server.Config("api")) != 1 || len(server.Domains("api")) != 1 || len(server.Buildpacks("api")) != 1 {
						return fmt.Errorf("not every resource was applied")
					}
					return nil
				}),
			},
		},
	})
}

func TestFailedCommandOutputIsMasked(t *testing.T) {
	server := dokkutest.New(t).SeedApp("api")
	server.HandlePrefix("--quiet config:import", dokkutest.Exit(1, "refused value s3cr3t-value"))
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeProviders(server),
		Steps: []resource.TestStep{
			{
				Config: `resource "dokku_config" "api" {
  app     = "api"
  restart = false
  config  = { SECRET = "s3cr3t-value" }
}`,
				ExpectError: regexp.MustCompile(`refused value \*\*\*`),
			},
		},
	})
}

func TestImperativeTypesRefuseImport(t *testing.T) {
	server := dokkutest.New(t).SeedApp("api")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeProviders(server),
		Steps: []resource.TestStep{
			{
				Config:        `resource "dokku_git_sync" "api" {}`,
				ResourceName:  "dokku_git_sync.api",
				ImportState:   true,
				ImportStateId: "dokku_git_sync[app=api]",
				ExpectError:   regexp.MustCompile(`nothing to import`),
			},
			{
				Config:        `resource "dokku_domains" "api" {}`,
				ResourceName:  "dokku_domains.api",
				ImportState:   true,
				ImportStateId: "dokku_app[app=api]",
				ExpectError:   regexp.MustCompile(`not a dokku_domains`),
			},
		},
	})
}

func TestDataSources(t *testing.T) {
	server := dokkutest.New(t).SeedApp("api").
		SeedConfig("api", map[string]string{"KEY": "value"}).
		SeedDomains("api", "api.example.com")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeProviders(server),
		Steps: []resource.TestStep{
			{
				Config: `
data "dokku_config" "api" { app = "api" }
data "dokku_domains" "api" { app = "api" }`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.dokku_config.api", "config.KEY", "value"),
					resource.TestCheckResourceAttr("data.dokku_config.api", "id", "dokku_config[app=api]"),
					resource.TestCheckTypeSetElemAttr("data.dokku_domains.api", "domains.*", "api.example.com"),
				),
			},
			{
				Config:      `data "dokku_config" "missing" { app = "missing" }`,
				ExpectError: regexp.MustCompile(`does not exist`),
			},
		},
	})
}

func TestStorageEnsureHasNoReadOrTeardown(t *testing.T) {
	server := dokkutest.New(t).SeedApp("api")
	server.HandlePrefix("--quiet storage:ensure-directory", dokkutest.Respond(""))
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeProviders(server),
		Steps: []resource.TestStep{
			{
				Config: `resource "dokku_storage_ensure" "api" { app = "api" }`,
				Check: checkServer(func() error {
					if n := server.Count("--quiet storage:ensure-directory api"); n != 1 {
						return fmt.Errorf("storage:ensure-directory ran %d times", n)
					}
					return nil
				}),
			},
		},
		CheckDestroy: func(*terraform.State) error {
			// Destroy removes it from state and runs nothing on the server.
			if n := server.Count("--quiet storage:ensure-directory api"); n != 1 {
				return fmt.Errorf("storage:ensure-directory ran %d times", n)
			}
			return nil
		},
	})
}

func TestWriteOnlyFieldsKeepTheirValue(t *testing.T) {
	server := dokkutest.New(t).SeedDatastore("redis", "redis:7.2")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeProviders(server),
		Steps: []resource.TestStep{
			{
				// A refresh reads no password back, and the plan after it
				// stays empty.
				Config: `resource "dokku_service_create" "cache" {
  service  = "redis"
  name     = "cache"
  password = "s3cr3t"
}`,
				Check: resource.TestCheckResourceAttr("dokku_service_create.cache", "password", "s3cr3t"),
			},
		},
	})
}

func TestApplyReadsAgainWithoutObservedState(t *testing.T) {
	server := dokkutest.New(t).SeedDatastore("redis", "redis:7.2")
	// The read docket makes after the create fails once, so the apply comes
	// back without the service's state and the provider reads it again.
	var mu sync.Mutex
	failed := false
	runner := func(ctx context.Context, in sdk.ExecCommandInput) (sdk.ExecCommandResponse, error) {
		mu.Lock()
		fail := !failed && server.Count("--quiet redis:create cache") == 1 && strings.Join(in.Args, " ") == "--quiet redis:info cache --format json"
		if fail {
			failed = true
		}
		mu.Unlock()
		if fail {
			res := sdk.ExecCommandResponse{ExitCode: 1, Stderr: "temporarily unavailable"}
			return res, &sdk.ExecError{Response: res, Err: errors.New(res.Stderr), Ran: true}
		}
		return server.Run(ctx, in)
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"dokku": providerserver.NewProtocol6WithError(&dokkuProvider{version: "test", runner: runner}),
		},
		Steps: []resource.TestStep{
			{
				Config: `resource "dokku_service_create" "cache" {
  service = "redis"
  name    = "cache"
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("dokku_service_create.cache", "image"),
					checkServer(func() error {
						if !failed {
							return fmt.Errorf("the read after the create never failed")
						}
						// The failed read never reached the fake; the one
						// after it is the provider's own.
						if n := server.Count("--quiet redis:info cache --format json"); n < 1 {
							return fmt.Errorf("the provider did not read the service again")
						}
						return nil
					}),
				),
			},
		},
	})
}
