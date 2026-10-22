package provider

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/dokku/docket/sdk"
	"github.com/dokku/docket/sdk/dokkutest"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// These tests run Terraform against the provider with every dokku command
// answered by docket's stateful fake, so they need the terraform binary but
// no server.

// fakeProviders returns provider factories whose commands all go to server.
func fakeProviders(server *dokkutest.Server) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"dokku": providerserver.NewProtocol6WithError(&dokkuProvider{version: "test", runner: server.Run}),
	}
}

// outOfBand runs a docket task against the fake directly, the way someone
// changing the server outside Terraform would.
func outOfBand(t *testing.T, server *dokkutest.Server, typeKey string, values map[string]any) {
	t.Helper()
	task, err := sdk.TaskFromValues(typeKey, values)
	if err != nil {
		t.Fatal(err)
	}
	if out := task.Execute(server.Context(context.Background())); out.Error != nil {
		t.Fatalf("%s: %v", typeKey, out.Error)
	}
}

// checkServer turns an assertion on the fake's state into a test check.
func checkServer(check func() error) resource.TestCheckFunc {
	return func(*terraform.State) error {
		return check()
	}
}

func TestAppLifecycle(t *testing.T) {
	server := dokkutest.New(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeProviders(server),
		Steps: []resource.TestStep{
			{
				Config: `resource "dokku_app" "api" { app = "api" }`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("dokku_app.api", "id", "dokku_app[app=api]"),
					checkServer(func() error {
						if !server.HasApp("api") {
							return fmt.Errorf("app api was not created")
						}
						return nil
					}),
				),
			},
			{
				ResourceName:      "dokku_app.api",
				ImportState:       true,
				ImportStateId:     "dokku_app[app=api]",
				ImportStateVerify: true,
			},
			{
				// Destroying the app out of band removes the resource, and
				// the plan recreates it.
				PreConfig: func() {
					outOfBand(t, server, "dokku_app", map[string]any{"app": "api", "state": "absent"})
				},
				Config:             `resource "dokku_app" "api" { app = "api" }`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
		CheckDestroy: func(*terraform.State) error {
			if server.HasApp("api") {
				return fmt.Errorf("app api still exists")
			}
			return nil
		},
	})
}

func TestConfigLifecycle(t *testing.T) {
	server := dokkutest.New(t).SeedApp("api")
	config := func(body string) string {
		return `resource "dokku_config" "api" {
  app     = "api"
  restart = false
  ` + body + `
}`
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeProviders(server),
		Steps: []resource.TestStep{
			{
				Config: config(`config = { KEY = "value", SECRET = "s3cr3t" }`),
				Check: checkServer(func() error {
					want := map[string]string{"KEY": "value", "SECRET": "s3cr3t"}
					if got := server.Config("api"); !maps.Equal(got, want) {
						return fmt.Errorf("config = %v, want %v", got, want)
					}
					return nil
				}),
			},
			{
				// A key added out of band is removed: the resource owns the
				// whole config.
				PreConfig: func() {
					outOfBand(t, server, "dokku_config", map[string]any{
						"app": "api", "config": map[string]any{"EXTRA": "1"}, "restart": false,
					})
				},
				Config: config(`config = { KEY = "value", SECRET = "s3cr3t" }`),
				Check: checkServer(func() error {
					if _, ok := server.Config("api")["EXTRA"]; ok {
						return fmt.Errorf("EXTRA was not removed")
					}
					return nil
				}),
			},
			{
				// Every key unset out of band plans an update, not a create.
				PreConfig: func() {
					outOfBand(t, server, "dokku_config", map[string]any{"app": "api", "state": "clear", "restart": false})
				},
				Config:             config(`config = { KEY = "value", SECRET = "s3cr3t" }`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				// Leaving config out declares it empty.
				Config: config(``),
				Check: checkServer(func() error {
					if got := server.Config("api"); len(got) != 0 {
						return fmt.Errorf("config = %v, want empty", got)
					}
					return nil
				}),
			},
			{
				ResourceName:            "dokku_config.api",
				ImportState:             true,
				ImportStateId:           "dokku_config[app=api]",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"restart"},
			},
		},
	})
}

func TestBuildpacksKeepOrder(t *testing.T) {
	server := dokkutest.New(t).SeedApp("api")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeProviders(server),
		Steps: []resource.TestStep{
			{
				Config: `resource "dokku_buildpacks" "api" {
  app        = "api"
  buildpacks = ["https://github.com/heroku/heroku-buildpack-nodejs.git", "https://github.com/heroku/heroku-buildpack-ruby.git"]
}`,
				Check: checkServer(func() error {
					want := []string{"https://github.com/heroku/heroku-buildpack-nodejs.git", "https://github.com/heroku/heroku-buildpack-ruby.git"}
					if got := server.Buildpacks("api"); !slices.Equal(got, want) {
						return fmt.Errorf("buildpacks = %v, want %v", got, want)
					}
					return nil
				}),
			},
			{
				ResourceName:      "dokku_buildpacks.api",
				ImportState:       true,
				ImportStateId:     "dokku_buildpacks[app=api]",
				ImportStateVerify: true,
			},
		},
		CheckDestroy: func(*terraform.State) error {
			if got := server.Buildpacks("api"); len(got) != 0 {
				return fmt.Errorf("buildpacks = %v after destroy", got)
			}
			return nil
		},
	})
}

func TestPsScaleClearsOnDestroy(t *testing.T) {
	server := dokkutest.New(t).SeedApp("api")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeProviders(server),
		Steps: []resource.TestStep{
			{
				Config: `resource "dokku_ps_scale" "api" {
  app         = "api"
  skip_deploy = true
  scale       = { web = 2, worker = 0 }
}`,
				Check: checkServer(func() error {
					if got := server.Scale("api")["web"]; got != 2 {
						return fmt.Errorf("web = %d, want 2", got)
					}
					return nil
				}),
			},
		},
		CheckDestroy: func(*terraform.State) error {
			if got := server.Scale("api")["web"]; got == 2 {
				return fmt.Errorf("scale was not reset: %v", server.Scale("api"))
			}
			return nil
		},
	})
}
