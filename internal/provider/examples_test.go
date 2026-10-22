package provider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dokku/docket/sdk/dokkutest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestGeneratedExamplesPlan plans every generated example, so each one is a
// configuration the provider accepts: its attributes exist, and it passes the
// task's own checks. Planning new resources sends no command to the server.
func TestGeneratedExamplesPlan(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "examples", "resources", "*", "resource.tf"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no examples; run go generate")
	}
	for _, file := range files {
		config, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(filepath.Base(filepath.Dir(file)), func(t *testing.T) {
			t.Parallel()
			server := dokkutest.New(t)
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: fakeProviders(server),
				Steps: []resource.TestStep{
					{
						Config:             string(config),
						PlanOnly:           true,
						ExpectNonEmptyPlan: true,
					},
				},
			})
		})
	}
}
