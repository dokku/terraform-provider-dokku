package provider

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/dokku/docket/sdk"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestEveryTaskTypeBuildsASchema(t *testing.T) {
	ctx := context.Background()
	types, err := taskTypes()
	if err != nil {
		t.Fatal(err)
	}
	if len(types) != len(sdk.TaskTypes()) {
		t.Fatalf("%d types mapped, docket has %d", len(types), len(sdk.TaskTypes()))
	}
	for _, tt := range types {
		r := newTaskResource(tt)()
		var resp resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("%s: %v", tt.Key, resp.Diagnostics)
			continue
		}
		if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
			t.Errorf("%s: %v", tt.Key, diags)
		}
		for _, f := range tt.Fields() {
			if _, ok := resp.Schema.Attributes[f.Name]; !ok {
				t.Errorf("%s.%s has no attribute", tt.Key, f.Name)
			}
		}
		if _, ok := resp.Schema.Attributes[stateField]; ok {
			t.Errorf("%s exposes state", tt.Key)
		}
		if !tt.Readable() {
			continue
		}
		d := newTaskDataSource(tt)()
		var dresp datasource.SchemaResponse
		d.Schema(ctx, datasource.SchemaRequest{}, &dresp)
		if dresp.Diagnostics.HasError() {
			t.Errorf("%s data source: %v", tt.Key, dresp.Diagnostics)
			continue
		}
		if diags := dresp.Schema.ValidateImplementation(ctx); diags.HasError() {
			t.Errorf("%s data source: %v", tt.Key, diags)
		}
	}
}

func TestLifecyclesComeFromTheCatalog(t *testing.T) {
	types, err := taskTypes()
	if err != nil {
		t.Fatal(err)
	}
	var noTeardown, unreadable []string
	for _, tt := range types {
		task, err := sdk.NewTask(tt.Key)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := tt.Lifecycle(), sdk.TaskLifecycle(task); got != want {
			t.Errorf("%s: lifecycle %+v, sdk says %+v", tt.Key, got, want)
		}
		if !tt.HasTeardown() {
			noTeardown = append(noTeardown, tt.Key)
		}
		if !tt.Readable() {
			unreadable = append(unreadable, tt.Key)
		}
	}
	want := []string{"dokku_app_clone", "dokku_git_from_archive", "dokku_git_from_image", "dokku_git_sync", "dokku_storage_ensure"}
	if !slices.Equal(noTeardown, want) {
		t.Errorf("types without teardown = %v, want %v", noTeardown, want)
	}
	if !slices.Equal(unreadable, want) {
		t.Errorf("unreadable types = %v, want %v", unreadable, want)
	}
}

func TestCollectionsAreAuthoritative(t *testing.T) {
	for _, key := range []string{"dokku_config", "dokku_domains", "dokku_ports", "dokku_buildpacks", "dokku_ps_scale"} {
		tt, err := lookupTaskType(key)
		if err != nil {
			t.Fatal(err)
		}
		if got := tt.Lifecycle(); got.Apply != sdk.StateSet || got.Teardown != sdk.StateClear {
			t.Errorf("%s lifecycle = %+v, want set/clear", key, got)
		}
	}
}

// attribute returns one resource attribute of a type.
func attribute(t *testing.T, key, name string) rschema.Attribute {
	t.Helper()
	tt, err := lookupTaskType(key)
	if err != nil {
		t.Fatal(err)
	}
	s, err := resourceSchema(context.Background(), tt)
	if err != nil {
		t.Fatal(err)
	}
	a, ok := s.Attributes[name]
	if !ok {
		t.Fatalf("%s has no attribute %s", key, name)
	}
	return a
}

func TestAttributeModes(t *testing.T) {
	cases := []struct {
		key, name                    string
		required, optional, computed bool
	}{
		// Identity keys.
		{"dokku_app", "app", true, false, false},
		{"dokku_domains", "global", false, true, false},
		// Identity collections: leaving one out declares it empty.
		{"dokku_config", "config", false, true, false},
		{"dokku_ps_scale", "scale", false, true, false},
		// A default.
		{"dokku_config", "restart", false, true, true},
		// Filled from the server when left out.
		{"dokku_service_create", "image", false, true, true},
		// Write-only: a read never fills it.
		{"dokku_service_create", "password", false, true, false},
	}
	for _, c := range cases {
		a := attribute(t, c.key, c.name)
		if a.IsRequired() != c.required || a.IsOptional() != c.optional || a.IsComputed() != c.computed {
			t.Errorf("%s.%s: required=%v optional=%v computed=%v, want %v %v %v",
				c.key, c.name, a.IsRequired(), a.IsOptional(), a.IsComputed(), c.required, c.optional, c.computed)
		}
	}
}

func TestListsAndSets(t *testing.T) {
	if _, ok := attribute(t, "dokku_buildpacks", "buildpacks").(rschema.ListAttribute); !ok {
		t.Error("buildpacks is not a list")
	}
	if _, ok := attribute(t, "dokku_domains", "domains").(rschema.SetAttribute); !ok {
		t.Error("domains is not a set")
	}
	if _, ok := attribute(t, "dokku_ports", "port_mappings").(rschema.SetNestedAttribute); !ok {
		t.Error("port_mappings is not a set of objects")
	}
	if _, ok := attribute(t, "dokku_config", "preserve").(rschema.SetAttribute); !ok {
		t.Error("preserve is not a set")
	}
}

func TestForbiddenFieldsAreLeftOut(t *testing.T) {
	tt, err := lookupTaskType("dokku_ps_scale")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range tt.Schema.Fields {
		if slices.Contains(f.ForbiddenWhen, string(sdk.StateSet)) && slices.ContainsFunc(tt.Fields(), func(g sdk.FieldSchema) bool { return g.Name == f.Name }) {
			t.Errorf("%s is forbidden under set but exposed", f.Name)
		}
	}
}

func TestCheckFieldSchemaRefusesUnmappableLists(t *testing.T) {
	unordered := false
	err := checkFieldSchema("dokku_x", "items", sdk.FieldSchema{Name: "items", Type: sdk.TypeList, Ordered: &unordered, Item: &sdk.ItemSchema{Type: sdk.TypeString}})
	if err == nil {
		t.Error("an unordered list that is neither unique nor a collection was accepted")
	}
	err = checkFieldSchema("dokku_x", "items", sdk.FieldSchema{Name: "items", Type: sdk.TypeList, Item: &sdk.ItemSchema{Type: sdk.TypeString}})
	if err == nil {
		t.Error("a list without ordered was accepted")
	}
}

func TestDescriptionsUseTheApplyState(t *testing.T) {
	tt, err := lookupTaskType("dokku_domains")
	if err != nil {
		t.Fatal(err)
	}
	f, _ := tt.Field("domains")
	if desc := fieldDescription(f, tt.Lifecycle().Apply); strings.Contains(desc, "when state is") {
		t.Errorf("domains description mentions other states: %q", desc)
	}
	if desc := taskDescription(tt, tt.Lifecycle().Apply); !strings.HasPrefix(desc, tt.Schema.Synopsis) {
		t.Errorf("description does not start with the synopsis: %q", desc)
	}
}

// configValue builds a resource configuration value from attribute values,
// with every other attribute null.
func configValue(t *testing.T, key string, attrs map[string]tftypes.Value) tftypes.Value {
	t.Helper()
	tt, err := lookupTaskType(key)
	if err != nil {
		t.Fatal(err)
	}
	s, err := resourceSchema(context.Background(), tt)
	if err != nil {
		t.Fatal(err)
	}
	typ := objectType(context.Background(), s)
	out := map[string]tftypes.Value{}
	for name, at := range typ.AttributeTypes {
		if v, ok := attrs[name]; ok {
			out[name] = v
			continue
		}
		out[name] = tftypes.NewValue(at, nil)
	}
	return tftypes.NewValue(typ, out)
}

func TestValidateReportsEveryViolation(t *testing.T) {
	tt, _ := lookupTaskType("dokku_domains")
	var diags diag.Diagnostics
	// app and global are mutually exclusive.
	validateTask(tt, configValue(t, "dokku_domains", map[string]tftypes.Value{
		"app":    tftypes.NewValue(tftypes.String, "api"),
		"global": tftypes.NewValue(tftypes.Bool, true),
	}), &diags)
	if !diags.HasError() {
		t.Fatal("app with global was accepted")
	}
}

func TestValidateSkipsRulesOnUnknownValues(t *testing.T) {
	tt, _ := lookupTaskType("dokku_domains")
	var diags diag.Diagnostics
	// The app comes from another resource, so whether exactly one of app and
	// global is set is not known yet.
	validateTask(tt, configValue(t, "dokku_domains", map[string]tftypes.Value{
		"app": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	}), &diags)
	if diags.HasError() {
		t.Fatalf("an unknown app was refused: %v", diags)
	}

	diags = nil
	validateTask(tt, configValue(t, "dokku_domains", nil), &diags)
	if !diags.HasError() {
		t.Fatal("neither app nor global was accepted")
	}
}

func TestValidateRunsTheTasksValueChecks(t *testing.T) {
	tt, _ := lookupTaskType("dokku_apps_property")
	var diags diag.Diagnostics
	validateTask(tt, configValue(t, "dokku_apps_property", map[string]tftypes.Value{
		"app":      tftypes.NewValue(tftypes.String, "api"),
		"property": tftypes.NewValue(tftypes.String, "no-such-property"),
		"value":    tftypes.NewValue(tftypes.String, "1"),
	}), &diags)
	if !diags.HasError() {
		t.Fatal("an unknown property name was accepted")
	}
	if detail := diags.Errors()[0].Detail(); !strings.Contains(detail, "no-such-property") {
		t.Errorf("the error does not name the property: %q", detail)
	}
}
