package provider

import (
	"strings"

	"github.com/dokku/docket/sdk"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// inputValidator is the value check every docket task runs before it plans.
type inputValidator interface {
	Validate() error
}

// validateTask checks a configuration against the task's own rules without
// contacting a server. Every declared constraint is checked at once, with the
// values Terraform does not know yet passed as unknown, so the catalog stays
// the only copy of each rule. Once every value is known, the task's own
// Validate checks what depends on a value, such as enums, formats and
// property names.
func validateTask(t *taskType, config tftypes.Value, diags *diag.Diagnostics) {
	values, unknown, err := toValues(t.Fields(), config)
	if err != nil {
		diags.AddError("Invalid configuration", err.Error())
		return
	}
	task, err := buildTask(t, values, t.Lifecycle().Apply)
	if err != nil {
		diags.AddError("Invalid configuration", err.Error())
		return
	}
	violations := sdk.ConstraintViolations(task, sdk.WithUnknown(unknown...))
	for _, v := range violations {
		if root := violationRoot(v.Field); root != "" {
			diags.AddAttributeError(path.Root(root), "Invalid configuration", v.Message)
			continue
		}
		diags.AddError("Invalid configuration", v.Message)
	}
	if len(violations) > 0 || len(unknown) > 0 {
		return
	}
	if validator, ok := task.(inputValidator); ok {
		if err := validator.Validate(); err != nil {
			diags.AddError("Invalid configuration", err.Error())
		}
	}
}

// violationRoot is the top-level attribute a violation's recipe key path
// names, such as users for `users[1].hash`. Elements of a set have no stable
// index in Terraform, so the error is reported on the whole attribute and the
// message keeps the full path.
func violationRoot(field string) string {
	if i := strings.IndexAny(field, "[."); i >= 0 {
		return field[:i]
	}
	return field
}
