package provider

import (
	"context"
	"fmt"

	"github.com/dokku/docket/sdk"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// providerData is what Configure hands every resource and data source: one
// session shared by every call, so its version cache, lock table and masker
// span the whole run.
type providerData struct {
	session *sdk.Session
	target  sdk.Target

	// runner replaces dokku in unit tests. Only test code sets it.
	runner sdk.ExecRunner
}

// context returns the context one provider call runs dokku under: the
// request's cancellation, with the session's target, masker, locks and
// version gate.
func (d *providerData) context(ctx context.Context) context.Context {
	ctx = d.session.Context(ctx, d.target)
	if d.runner != nil {
		ctx = sdk.ContextWithRunner(ctx, d.runner)
	}
	return ctx
}

// addDiagnostics appends docket's diagnostics, already masked, to diags.
// warnOnly reports errors as warnings, for a failure that does not fail the
// operation.
func addDiagnostics(diags *diag.Diagnostics, entries []sdk.Diagnostic, warnOnly bool) {
	for _, d := range entries {
		if d.Severity == sdk.DiagnosticError && !warnOnly {
			diags.AddError(d.Summary, d.Detail)
			continue
		}
		diags.AddWarning(d.Summary, d.Detail)
	}
}

// hasErrors reports whether docket's diagnostics include an error.
func hasErrors(entries []sdk.Diagnostic) bool {
	for _, d := range entries {
		if d.Severity == sdk.DiagnosticError {
			return true
		}
	}
	return false
}

// buildTask builds a task of type t in state from a value map.
func buildTask(t *taskType, values map[string]any, state sdk.State) (sdk.Task, error) {
	in := make(map[string]any, len(values)+1)
	for key, value := range values {
		in[key] = value
	}
	in[stateField] = string(state)
	task, err := sdk.TaskFromValues(t.Key, in)
	if err != nil {
		return nil, fmt.Errorf("build the %s task: %w", t.Key, err)
	}
	return task, nil
}

// observedValues reads a task back as a value map without its state.
func observedValues(task sdk.Task) map[string]any {
	values := sdk.TaskValues(task)
	delete(values, stateField)
	return values
}
