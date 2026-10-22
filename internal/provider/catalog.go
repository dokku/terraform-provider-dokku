package provider

import (
	"fmt"
	"slices"
	"sync"

	"github.com/dokku/docket/sdk"
)

// stateField is the recipe key of every task's state. The provider drives the
// state itself, so it is never an attribute.
const stateField = "state"

// taskType is one docket task type as the provider exposes it.
type taskType struct {
	// Key is the docket type key, which is also the resource and data source
	// type name.
	Key string

	// Schema is the type's catalog entry.
	Schema sdk.TaskSchema
}

// Lifecycle is the pair of states the provider drives the task through.
func (t *taskType) Lifecycle() sdk.LifecycleSchema {
	return t.Schema.Lifecycle
}

// HasTeardown reports whether destroying the resource changes the server.
func (t *taskType) HasTeardown() bool {
	return t.Schema.Lifecycle.Teardown != ""
}

// Readable reports whether the provider can read the resource back from the
// server: the task declares a resource and its probe can read it.
func (t *taskType) Readable() bool {
	return !t.Schema.Imperative && t.Schema.Probe.Status != sdk.ProbeUnsupported
}

// Fields returns the fields exposed as resource attributes: every field but
// the state, and those the apply state forbids.
func (t *taskType) Fields() []sdk.FieldSchema {
	apply := string(t.Schema.Lifecycle.Apply)
	var fields []sdk.FieldSchema
	for _, f := range t.Schema.Fields {
		if f.Name == stateField || slices.Contains(f.ForbiddenWhen, apply) {
			continue
		}
		fields = append(fields, f)
	}
	return fields
}

// Field returns the named field from the catalog entry.
func (t *taskType) Field(name string) (sdk.FieldSchema, bool) {
	for _, f := range t.Schema.Fields {
		if f.Name == name {
			return f, true
		}
	}
	return sdk.FieldSchema{}, false
}

// loadCatalog reads the docket catalog once and indexes it by type key. It
// refuses a catalog the provider cannot map, so a docket upgrade that adds an
// unknown shape fails loudly instead of producing a wrong schema.
var loadCatalog = sync.OnceValues(func() (map[string]*taskType, error) {
	catalog, err := sdk.Catalog()
	if err != nil {
		return nil, fmt.Errorf("read the docket task catalog: %w", err)
	}
	types := make(map[string]*taskType, len(catalog.Tasks))
	for _, schema := range catalog.Tasks {
		if err := checkTaskSchema(schema); err != nil {
			return nil, err
		}
		types[schema.Type] = &taskType{Key: schema.Type, Schema: schema}
	}
	return types, nil
})

// taskTypes returns every catalog type, sorted by key.
func taskTypes() ([]*taskType, error) {
	types, err := loadCatalog()
	if err != nil {
		return nil, err
	}
	out := make([]*taskType, 0, len(types))
	for _, key := range sdk.TaskTypes() {
		if t, ok := types[key]; ok {
			out = append(out, t)
		}
	}
	return out, nil
}

// lookupTaskType returns one catalog type.
func lookupTaskType(key string) (*taskType, error) {
	types, err := loadCatalog()
	if err != nil {
		return nil, err
	}
	t, ok := types[key]
	if !ok {
		return nil, fmt.Errorf("docket has no task type %q", key)
	}
	return t, nil
}

// checkTaskSchema refuses a task the provider cannot map onto Terraform.
func checkTaskSchema(schema sdk.TaskSchema) error {
	if schema.Lifecycle.Apply == "" {
		return fmt.Errorf("%s: the catalog declares no apply state", schema.Type)
	}
	hasState := false
	for _, f := range schema.Fields {
		if f.Name == stateField {
			hasState = true
		}
		if err := checkFieldSchema(schema.Type, f.Name, f); err != nil {
			return err
		}
	}
	if !hasState {
		return fmt.Errorf("%s: the task has no state field", schema.Type)
	}
	return nil
}

// checkFieldSchema refuses a field whose shape has no Terraform mapping.
func checkFieldSchema(typeKey, path string, f sdk.FieldSchema) error {
	switch f.Type {
	case sdk.TypeString, sdk.TypeBool, sdk.TypeInt, sdk.TypeFloat:
		return nil
	case sdk.TypeDict:
		if f.Item == nil {
			return fmt.Errorf("%s.%s: a dict without an item type", typeKey, path)
		}
		switch f.Item.Type {
		case sdk.TypeString, sdk.TypeBool, sdk.TypeInt, sdk.TypeFloat, sdk.TypeAny:
			return nil
		}
		return fmt.Errorf("%s.%s: a dict of %s is not supported", typeKey, path, f.Item.Type)
	case sdk.TypeList:
		if f.Item == nil {
			return fmt.Errorf("%s.%s: a list without an item type", typeKey, path)
		}
		if f.Ordered == nil {
			return fmt.Errorf("%s.%s: the catalog does not say whether the list is ordered", typeKey, path)
		}
		if !*f.Ordered && !f.Unique && f.Identity != sdk.IdentityRoleCollection {
			return fmt.Errorf("%s.%s: an unordered list that is neither an identity collection nor unique cannot be a set", typeKey, path)
		}
		switch f.Item.Type {
		case sdk.TypeString, sdk.TypeBool, sdk.TypeInt, sdk.TypeFloat:
			return nil
		case sdk.TypeObject:
			for _, inner := range f.Item.Fields {
				if err := checkFieldSchema(typeKey, path+"."+inner.Name, inner); err != nil {
					return err
				}
			}
			return nil
		}
		return fmt.Errorf("%s.%s: a list of %s is not supported", typeKey, path, f.Item.Type)
	}
	return fmt.Errorf("%s.%s: field type %q is not supported", typeKey, path, f.Type)
}
