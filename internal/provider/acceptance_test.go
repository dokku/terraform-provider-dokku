package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/dokku/docket/sdk"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// The TestAcc* tests run against a real Dokku server and only when TF_ACC is
// set. The provider reads its settings from the DOKKU_* environment
// variables, so CI runs them on the server it installed, with the local
// target and DOKKU_SUDO=true.

var accProviders = map[string]func() (tfprotov6.ProviderServer, error){
	"dokku": providerserver.NewProtocol6WithError(New("acc")()),
}

// accContext reads the server directly, for checks and prechecks.
func accContext() context.Context {
	return newSession(false).Context(context.Background(), sdk.Target{
		Host: os.Getenv("DOKKU_HOST"),
		Sudo: boolSetting(types.BoolNull(), "DOKKU_SUDO"),
	})
}

// requirePlugin skips a test when a plugin it needs is not installed.
func requirePlugin(t *testing.T, name string) {
	t.Helper()
	_, exists, err := sdk.ReadAddress(accContext(), fmt.Sprintf("dokku_plugin[name=%s]", name))
	if err != nil || !exists {
		t.Skipf("the %s plugin is not installed", name)
	}
}

func TestAccAppConfigDomainsScale(t *testing.T) {
	app := "tf-acc-app"
	config := func(level string) string {
		return fmt.Sprintf(`
resource "dokku_app" "app" { app = %[1]q }

resource "dokku_config" "app" {
  app     = dokku_app.app.app
  restart = false
  config  = { LOG_LEVEL = %[2]q, SECRET = "s3cr3t" }
}

resource "dokku_domains" "app" {
  app     = dokku_app.app.app
  domains = ["%[1]s.example.com"]
}

resource "dokku_ps_scale" "app" {
  app         = dokku_app.app.app
  skip_deploy = true
  scale       = { web = 1, worker = 0 }
}`, app, level)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: accProviders,
		Steps: []resource.TestStep{
			{
				Config: config("info"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("dokku_config.app", "config.LOG_LEVEL", "info"),
					resource.TestCheckResourceAttr("dokku_ps_scale.app", "scale.web", "1"),
				),
			},
			{
				Config: config("debug"),
				Check:  resource.TestCheckResourceAttr("dokku_config.app", "config.LOG_LEVEL", "debug"),
			},
			{
				// A key added outside Terraform is drift the resource owns.
				PreConfig: func() {
					task, _ := sdk.TaskFromValues("dokku_config", map[string]any{
						"app": app, "config": map[string]any{"EXTRA": "1"}, "restart": false,
					})
					if out := task.Execute(accContext()); out.Error != nil {
						t.Fatal(out.Error)
					}
				},
				Config:             config("debug"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{Config: config("debug")},
			{
				ResourceName:            "dokku_config.app",
				ImportState:             true,
				ImportStateId:           fmt.Sprintf("dokku_config[app=%s]", app),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"restart"},
			},
			{
				ResourceName:      "dokku_domains.app",
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("dokku_domains[app=%s]", app),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDataSourcesAndExport(t *testing.T) {
	app := "tf-acc-data"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: accProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "dokku_app" "app" { app = %[1]q }

resource "dokku_config" "app" {
  app     = dokku_app.app.app
  restart = false
  config  = { KEY = "value" }
}

data "dokku_config" "app" {
  app        = dokku_config.app.app
  depends_on = [dokku_config.app]
}

data "dokku_export" "app" {
  apps       = [dokku_app.app.app]
  depends_on = [dokku_config.app]
}`, app),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.dokku_config.app", "config.KEY", "value"),
					resource.TestMatchResourceAttr("data.dokku_export.app", "recipe", regexp.MustCompile(app)),
				),
			},
		},
	})
}

func TestAccServiceFamily(t *testing.T) {
	requirePlugin(t, "redis")
	app := "tf-acc-svc"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: accProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "dokku_app" "app" { app = %[1]q }

resource "dokku_service_create" "cache" {
  service = "redis"
  name    = "tf-acc-cache"
}

resource "dokku_service_link" "app" {
  app     = dokku_app.app.app
  service = dokku_service_create.cache.service
  name    = dokku_service_create.cache.name
}`, app),
				Check: resource.TestCheckResourceAttrSet("dokku_service_create.cache", "image"),
			},
			{
				ResourceName:            "dokku_service_create.cache",
				ImportState:             true,
				ImportStateId:           "dokku_service_create[service=redis,name=tf-acc-cache]",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password", "root_password", "image_drift", "restart_apps"},
			},
		},
	})
}
