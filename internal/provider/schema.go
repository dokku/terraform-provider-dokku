package provider

import (
	"context"
	"fmt"
	gopath "path"
	"regexp"
	"runtime/debug"
	"slices"
	"strings"

	"github.com/dokku/docket/sdk"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/dynamicplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// idAttribute is the computed attribute holding a resource's identity
// address. No docket field is named id.
const idAttribute = "id"

// attrMode is how Terraform treats one attribute.
type attrMode int

const (
	modeRequired attrMode = iota
	modeOptional
	modeOptionalComputed
	modeComputed
)

// resourceMode decides how a field is set in a resource. apply is the state
// the provider drives the task in.
//
//   - A field the task requires, always or in the apply state, is required.
//   - An identity collection is optional: leaving it out declares it empty.
//   - An identity key is optional, or optional with its default.
//   - Any other field with a default is optional and computed, holding the
//     default when it is left out.
//   - A write-only or apply-only field is optional, since a read never fills
//     it.
//   - Every other field is optional and computed: docket fills a field the
//     task leaves unset from the server, so leaving it out keeps the server's
//     value rather than planning to remove it.
func resourceMode(f sdk.FieldSchema, apply sdk.State) attrMode {
	switch {
	case f.Required || slices.Contains(f.RequiredWhen, string(apply)):
		return modeRequired
	case f.Identity == sdk.IdentityRoleCollection:
		return modeOptional
	case f.Default != nil:
		return modeOptionalComputed
	case f.Identity == sdk.IdentityRoleKey, f.WriteOnly, f.ApplyOnly:
		return modeOptional
	}
	return modeOptionalComputed
}

// keptFromConfig reports whether a field's value always comes from the
// configuration rather than the server, because a read never returns it.
func keptFromConfig(f sdk.FieldSchema) bool {
	return f.WriteOnly || f.ApplyOnly
}

// fieldDescription is a field's description followed by the note docket's
// reference page appends to it, as it reads in state.
func fieldDescription(f sdk.FieldSchema, state sdk.State) string {
	desc := f.Description
	if f.Note != nil {
		if suffix, ok := f.Note.States[string(state)]; ok {
			desc += suffix
		} else if len(f.Note.States) == 0 {
			desc += f.Note.Suffix
		}
	}
	return absoluteLinks(strings.TrimSpace(desc))
}

// taskDescription is the task's synopsis followed by each of its rendered
// notes, as they read in state.
func taskDescription(t *taskType, state sdk.State) string {
	parts := []string{strings.TrimSpace(t.Schema.Synopsis)}
	for _, note := range t.Schema.Notes {
		text := note.Markdown
		if len(note.States) > 0 {
			text = note.States[string(state)]
		}
		if text = strings.TrimSpace(text); text != "" {
			parts = append(parts, absoluteLinks(demoteHeadings(text)))
		}
	}
	return strings.Join(parts, "\n\n")
}

// relativeLink matches a markdown link whose target is a relative path.
var relativeLink = regexp.MustCompile(`\]\(([^)#:][^):]*)\)`)

// absoluteLinks points docket's relative links, which are relative to its
// docs/tasks directory, at that file in the docket release the provider is
// built with.
func absoluteLinks(markdown string) string {
	return relativeLink.ReplaceAllStringFunc(markdown, func(match string) string {
		target := relativeLink.FindStringSubmatch(match)[1]
		return "](" + docketDocsURL + gopath.Join("docs/tasks", target) + ")"
	})
}

// docketDocsURL is the root of the docket repository at the release the
// provider is built with.
var docketDocsURL = func() string {
	ref := "main"
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/dokku/docket" && dep.Version != "" && dep.Version != "(devel)" {
				ref = dep.Version
			}
		}
	}
	return "https://github.com/dokku/docket/blob/" + ref + "/"
}()

// demoteHeadings moves docket's level-two note headings down to level three,
// so they nest under the registry page's own sections.
func demoteHeadings(markdown string) string {
	lines := strings.Split(markdown, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "#") {
			lines[i] = "#" + line
		}
	}
	return strings.Join(lines, "\n")
}

// resourceSchema builds the resource schema for a task type.
func resourceSchema(ctx context.Context, t *taskType) (rschema.Schema, error) {
	apply := t.Lifecycle().Apply
	attrs := map[string]rschema.Attribute{
		idAttribute: rschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The resource's identity address, such as `" + t.Key + "[app=api]`.",
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
	}
	for _, f := range t.Fields() {
		a, err := resourceAttribute(ctx, f, apply, false)
		if err != nil {
			return rschema.Schema{}, fmt.Errorf("%s.%s: %w", t.Key, f.Name, err)
		}
		attrs[f.Name] = a
	}
	return rschema.Schema{
		Description:         strings.TrimSpace(t.Schema.Synopsis),
		MarkdownDescription: taskDescription(t, apply),
		DeprecationMessage:  t.Schema.Deprecation,
		Attributes:          attrs,
	}, nil
}

// resourceAttribute maps one field onto a resource attribute. nested is true
// for a field of a list element, where plan modifiers that read prior state by
// path do not apply.
func resourceAttribute(ctx context.Context, f sdk.FieldSchema, apply sdk.State, nested bool) (rschema.Attribute, error) {
	mode := resourceMode(f, apply)
	required := mode == modeRequired
	optional := mode == modeOptional || mode == modeOptionalComputed
	computed := mode == modeOptionalComputed
	desc := fieldDescription(f, apply)
	replace := !nested && f.Identity == sdk.IdentityRoleKey
	keepState := !nested && computed && f.Default == nil

	switch f.Type {
	case sdk.TypeString:
		a := rschema.StringAttribute{Required: required, Optional: optional, Computed: computed, Sensitive: f.Sensitive, MarkdownDescription: desc}
		if len(f.Choices) > 0 {
			a.Validators = []validator.String{stringvalidator.OneOf(f.Choices...)}
		}
		if v, ok := f.Default.(string); ok {
			a.Default = stringdefault.StaticString(v)
		}
		if replace {
			a.PlanModifiers = append(a.PlanModifiers, stringplanmodifier.RequiresReplace())
		}
		if keepState {
			a.PlanModifiers = append(a.PlanModifiers, stringplanmodifier.UseStateForUnknown())
		}
		return a, nil
	case sdk.TypeBool:
		a := rschema.BoolAttribute{Required: required, Optional: optional, Computed: computed, Sensitive: f.Sensitive, MarkdownDescription: desc}
		if v, ok := f.Default.(bool); ok {
			a.Default = booldefault.StaticBool(v)
		}
		if replace {
			a.PlanModifiers = append(a.PlanModifiers, boolplanmodifier.RequiresReplace())
		}
		if keepState {
			a.PlanModifiers = append(a.PlanModifiers, boolplanmodifier.UseStateForUnknown())
		}
		return a, nil
	case sdk.TypeInt:
		a := rschema.Int64Attribute{Required: required, Optional: optional, Computed: computed, Sensitive: f.Sensitive, MarkdownDescription: desc}
		if v, ok := f.Default.(int64); ok {
			a.Default = int64default.StaticInt64(v)
		}
		if replace {
			a.PlanModifiers = append(a.PlanModifiers, int64planmodifier.RequiresReplace())
		}
		if keepState {
			a.PlanModifiers = append(a.PlanModifiers, int64planmodifier.UseStateForUnknown())
		}
		return a, nil
	case sdk.TypeFloat:
		a := rschema.Float64Attribute{Required: required, Optional: optional, Computed: computed, Sensitive: f.Sensitive, MarkdownDescription: desc}
		if v, ok := f.Default.(float64); ok {
			a.Default = float64default.StaticFloat64(v)
		}
		if replace {
			a.PlanModifiers = append(a.PlanModifiers, float64planmodifier.RequiresReplace())
		}
		if keepState {
			a.PlanModifiers = append(a.PlanModifiers, float64planmodifier.UseStateForUnknown())
		}
		return a, nil
	case sdk.TypeDict:
		if f.Item.Type == sdk.TypeAny {
			a := rschema.DynamicAttribute{Required: required, Optional: optional, Computed: computed, Sensitive: f.Sensitive, MarkdownDescription: desc}
			if keepState {
				a.PlanModifiers = []planmodifier.Dynamic{dynamicplanmodifier.UseStateForUnknown()}
			}
			return a, nil
		}
		a := rschema.MapAttribute{Required: required, Optional: optional, Computed: computed, Sensitive: f.Sensitive, MarkdownDescription: desc, ElementType: scalarType(f.Item.Type)}
		if keepState {
			a.PlanModifiers = []planmodifier.Map{mapplanmodifier.UseStateForUnknown()}
		}
		return a, nil
	case sdk.TypeList:
		ordered := *f.Ordered
		if f.Item.Type == sdk.TypeObject {
			inner := map[string]rschema.Attribute{}
			for _, nf := range f.Item.Fields {
				a, err := resourceAttribute(ctx, nf, apply, true)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", nf.Name, err)
				}
				inner[nf.Name] = a
			}
			object := rschema.NestedAttributeObject{Attributes: inner}
			if ordered {
				a := rschema.ListNestedAttribute{Required: required, Optional: optional, Computed: computed, Sensitive: f.Sensitive, MarkdownDescription: desc, NestedObject: object}
				if keepState {
					a.PlanModifiers = []planmodifier.List{listplanmodifier.UseStateForUnknown()}
				}
				return a, nil
			}
			a := rschema.SetNestedAttribute{Required: required, Optional: optional, Computed: computed, Sensitive: f.Sensitive, MarkdownDescription: desc, NestedObject: object}
			if keepState {
				a.PlanModifiers = []planmodifier.Set{setplanmodifier.UseStateForUnknown()}
			}
			return a, nil
		}
		elem := scalarType(f.Item.Type)
		if ordered {
			a := rschema.ListAttribute{Required: required, Optional: optional, Computed: computed, Sensitive: f.Sensitive, MarkdownDescription: desc, ElementType: elem}
			if f.Default != nil {
				v, err := defaultValue(ctx, types.ListType{ElemType: elem}, f.Default)
				if err != nil {
					return nil, err
				}
				a.Default = listdefault.StaticValue(v.(types.List))
			}
			if keepState {
				a.PlanModifiers = []planmodifier.List{listplanmodifier.UseStateForUnknown()}
			}
			return a, nil
		}
		a := rschema.SetAttribute{Required: required, Optional: optional, Computed: computed, Sensitive: f.Sensitive, MarkdownDescription: desc, ElementType: elem}
		if f.Default != nil {
			v, err := defaultValue(ctx, types.SetType{ElemType: elem}, f.Default)
			if err != nil {
				return nil, err
			}
			a.Default = setdefault.StaticValue(v.(types.Set))
		}
		if keepState {
			a.PlanModifiers = []planmodifier.Set{setplanmodifier.UseStateForUnknown()}
		}
		return a, nil
	}
	return nil, fmt.Errorf("field type %q is not supported", f.Type)
}

// dataSourceSchema builds the data source schema for a readable task type.
// The identity keys are the inputs, and every field a read fills is computed.
func dataSourceSchema(t *taskType) (dschema.Schema, error) {
	apply := t.Lifecycle().Apply
	attrs := map[string]dschema.Attribute{
		idAttribute: dschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The resource's identity address.",
		},
	}
	for _, f := range t.Fields() {
		if keptFromConfig(f) {
			continue
		}
		a, err := dataSourceAttribute(f, apply)
		if err != nil {
			return dschema.Schema{}, fmt.Errorf("%s.%s: %w", t.Key, f.Name, err)
		}
		attrs[f.Name] = a
	}
	return dschema.Schema{
		MarkdownDescription: strings.TrimSpace(t.Schema.Synopsis),
		DeprecationMessage:  t.Schema.Deprecation,
		Attributes:          attrs,
	}, nil
}

// dataSourceAttribute maps one field onto a data source attribute. An
// identity key is an input, required when the task requires it; every other
// field is computed.
func dataSourceAttribute(f sdk.FieldSchema, apply sdk.State) (dschema.Attribute, error) {
	required, optional, computed := false, false, true
	if f.Identity == sdk.IdentityRoleKey {
		if f.Required {
			required, computed = true, false
		} else {
			optional = true
		}
	}
	desc := fieldDescription(f, apply)
	switch f.Type {
	case sdk.TypeString:
		return dschema.StringAttribute{Required: required, Optional: optional, Computed: computed, Sensitive: f.Sensitive, MarkdownDescription: desc}, nil
	case sdk.TypeBool:
		return dschema.BoolAttribute{Required: required, Optional: optional, Computed: computed, Sensitive: f.Sensitive, MarkdownDescription: desc}, nil
	case sdk.TypeInt:
		return dschema.Int64Attribute{Required: required, Optional: optional, Computed: computed, Sensitive: f.Sensitive, MarkdownDescription: desc}, nil
	case sdk.TypeFloat:
		return dschema.Float64Attribute{Required: required, Optional: optional, Computed: computed, Sensitive: f.Sensitive, MarkdownDescription: desc}, nil
	case sdk.TypeDict:
		if f.Item.Type == sdk.TypeAny {
			return dschema.DynamicAttribute{Computed: true, Sensitive: f.Sensitive, MarkdownDescription: desc}, nil
		}
		return dschema.MapAttribute{Computed: true, Sensitive: f.Sensitive, MarkdownDescription: desc, ElementType: scalarType(f.Item.Type)}, nil
	case sdk.TypeList:
		if f.Item.Type == sdk.TypeObject {
			inner := map[string]dschema.Attribute{}
			for _, nf := range f.Item.Fields {
				if keptFromConfig(nf) {
					continue
				}
				inner[nf.Name] = nestedDataSourceAttribute(nf, apply)
			}
			object := dschema.NestedAttributeObject{Attributes: inner}
			if *f.Ordered {
				return dschema.ListNestedAttribute{Computed: true, Sensitive: f.Sensitive, MarkdownDescription: desc, NestedObject: object}, nil
			}
			return dschema.SetNestedAttribute{Computed: true, Sensitive: f.Sensitive, MarkdownDescription: desc, NestedObject: object}, nil
		}
		if *f.Ordered {
			return dschema.ListAttribute{Computed: true, Sensitive: f.Sensitive, MarkdownDescription: desc, ElementType: scalarType(f.Item.Type)}, nil
		}
		return dschema.SetAttribute{Computed: true, Sensitive: f.Sensitive, MarkdownDescription: desc, ElementType: scalarType(f.Item.Type)}, nil
	}
	return nil, fmt.Errorf("field type %q is not supported", f.Type)
}

// nestedDataSourceAttribute maps a field of a list element onto a computed
// data source attribute.
func nestedDataSourceAttribute(f sdk.FieldSchema, apply sdk.State) dschema.Attribute {
	desc := fieldDescription(f, apply)
	switch f.Type {
	case sdk.TypeBool:
		return dschema.BoolAttribute{Computed: true, Sensitive: f.Sensitive, MarkdownDescription: desc}
	case sdk.TypeInt:
		return dschema.Int64Attribute{Computed: true, Sensitive: f.Sensitive, MarkdownDescription: desc}
	case sdk.TypeFloat:
		return dschema.Float64Attribute{Computed: true, Sensitive: f.Sensitive, MarkdownDescription: desc}
	case sdk.TypeList:
		if *f.Ordered {
			return dschema.ListAttribute{Computed: true, Sensitive: f.Sensitive, MarkdownDescription: desc, ElementType: scalarType(f.Item.Type)}
		}
		return dschema.SetAttribute{Computed: true, Sensitive: f.Sensitive, MarkdownDescription: desc, ElementType: scalarType(f.Item.Type)}
	case sdk.TypeDict:
		return dschema.MapAttribute{Computed: true, Sensitive: f.Sensitive, MarkdownDescription: desc, ElementType: scalarType(f.Item.Type)}
	}
	return dschema.StringAttribute{Computed: true, Sensitive: f.Sensitive, MarkdownDescription: desc}
}

// scalarType is the framework type of a scalar catalog type.
func scalarType(t string) attr.Type {
	switch t {
	case sdk.TypeBool:
		return types.BoolType
	case sdk.TypeInt:
		return types.Int64Type
	case sdk.TypeFloat:
		return types.Float64Type
	}
	return types.StringType
}

// defaultValue converts a typed catalog default into a framework value of t.
func defaultValue(ctx context.Context, t attr.Type, def any) (attr.Value, error) {
	tfv, err := goToTerraform(t.TerraformType(ctx), def)
	if err != nil {
		return nil, fmt.Errorf("default %v: %w", def, err)
	}
	return t.ValueFromTerraform(ctx, tfv)
}

// objectType returns the Terraform object type of a schema.
func objectType(ctx context.Context, s interface {
	Type() attr.Type
}) tftypes.Object {
	return s.Type().TerraformType(ctx).(tftypes.Object)
}
