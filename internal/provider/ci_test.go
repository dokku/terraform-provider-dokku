package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"golang.org/x/mod/modfile"
)

// TestSetupDokkuMatchesDocket keeps the acceptance job's server in step with
// the SDK: docket's setup-dokku action installs the Dokku and plugin
// versions its release was tested against, so its tag must be the docket
// version in go.mod.
func TestSetupDokkuMatchesDocket(t *testing.T) {
	gomod, err := os.ReadFile(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	mod, err := modfile.Parse("go.mod", gomod, nil)
	if err != nil {
		t.Fatal(err)
	}
	docket := ""
	for _, req := range mod.Require {
		if req.Mod.Path == "github.com/dokku/docket" {
			docket = req.Mod.Version
		}
	}
	if docket == "" {
		t.Fatal("go.mod does not require github.com/dokku/docket")
	}

	workflow, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "test.yml"))
	if err != nil {
		t.Fatal(err)
	}
	uses := regexp.MustCompile(`dokku/docket/\.github/actions/setup-dokku@(\S+)`).FindAllSubmatch(workflow, -1)
	if len(uses) == 0 {
		t.Fatal("the test workflow does not use docket's setup-dokku action")
	}
	for _, m := range uses {
		if tag := string(m[1]); tag != docket {
			t.Errorf("setup-dokku is pinned at %s, but go.mod requires docket %s", tag, docket)
		}
	}
}
