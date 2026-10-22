package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/dokku/docket/sdk"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	_ resource.ResourceWithConfigure      = (*taskResource)(nil)
	_ resource.ResourceWithImportState    = (*taskResource)(nil)
	_ resource.ResourceWithValidateConfig = (*taskResource)(nil)
)

// taskResource is the resource for one docket task type. Every type shares
// it: the schema and every value come from the catalog.
type taskResource struct {
	t    *taskType
	data *providerData
}

// newTaskResource returns the resource constructor for a task type.
func newTaskResource(t *taskType) func() resource.Resource {
	return func() resource.Resource {
		return &taskResource{t: t}
	}
}

func (r *taskResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.t.Key
}

func (r *taskResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s, err := resourceSchema(ctx, r.t)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported task schema", err.Error())
		return
	}
	resp.Schema = s
}

func (r *taskResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*providerData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("expected *providerData, got %T", req.ProviderData))
		return
	}
	r.data = data
}

func (r *taskResource) ValidateConfig(_ context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	validateTask(r.t, req.Config.Raw, &resp.Diagnostics)
}

func (r *taskResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.apply(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *taskResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.apply(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

// apply executes the task in its apply state and records the result. The
// state after the apply is the task as the server now holds it, so a value
// the plan left unknown takes the server's.
func (r *taskResource) apply(ctx context.Context, plan tfsdk.Plan, state *tfsdk.State, diags *diag.Diagnostics) {
	fields := r.t.Fields()
	values, _, err := toValues(fields, plan.Raw)
	if err != nil {
		diags.AddError("Invalid plan", err.Error())
		return
	}
	task, err := buildTask(r.t, values, r.t.Lifecycle().Apply)
	if err != nil {
		diags.AddError("Invalid plan", err.Error())
		return
	}

	ctx = r.data.context(ctx)
	out := task.Execute(ctx)
	entries := sdk.Diagnostics(ctx, out)
	addDiagnostics(diags, entries, false)
	if out.Error != nil || hasErrors(entries) {
		return
	}

	// A task with nothing to read records what it was given; any value the
	// plan left unknown becomes null. Any other task that came back without
	// its observed state is read once more, and failing that the apply still
	// stands.
	observedTask := out.Observed
	if observedTask == nil && r.t.Readable() {
		read, _, err := sdk.ReadTask(ctx, task)
		if err != nil {
			addDiagnostics(diags, sdk.Diagnostics(ctx, err), true)
		} else {
			observedTask = read
		}
	}
	observed := values
	if observedTask != nil {
		observed = observedValues(observedTask)
	}
	r.store(ctx, task, observed, plan.Raw, state, diags)
}

// store writes observed values into state, reconciled against prior.
func (r *taskResource) store(ctx context.Context, task sdk.Task, observed map[string]any, prior tftypes.Value, state *tfsdk.State, diags *diag.Diagnostics) {
	typ := objectType(ctx, state.Schema)
	id := tftypes.NewValue(tftypes.String, sdk.IdentityAddress(r.t.Key, task))
	raw, err := fromObserved(r.t.Fields(), typ, observed, prior, map[string]tftypes.Value{idAttribute: id})
	if err != nil {
		diags.AddError("Unable to record the resource state", err.Error())
		return
	}
	state.Raw = raw
}

func (r *taskResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// An imperative resource records that an operation ran; there is
	// nothing on the server to compare it with.
	if r.t.Schema.Imperative {
		return
	}
	values, _, err := toValues(r.t.Fields(), req.State.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Invalid state", err.Error())
		return
	}
	task, err := buildTask(r.t, values, r.t.Lifecycle().Apply)
	if err != nil {
		resp.Diagnostics.AddError("Invalid state", err.Error())
		return
	}

	ctx = r.data.context(ctx)
	read, exists, err := sdk.ReadTask(ctx, task)
	switch {
	case errors.Is(err, sdk.ErrObjectMissing):
		// The app or service the resource belongs to is gone, so the
		// resource is too; the next apply recreates it after its parent.
		resp.State.RemoveResource(ctx)
		return
	case errors.Is(err, sdk.ErrStateUnreadable):
		addDiagnostics(&resp.Diagnostics, sdk.Diagnostics(ctx, err), true)
		return
	case err != nil:
		addDiagnostics(&resp.Diagnostics, sdk.Diagnostics(ctx, err), false)
		return
	case !exists:
		resp.State.RemoveResource(ctx)
		return
	}
	r.store(ctx, task, observedValues(read), req.State.Raw, &resp.State, &resp.Diagnostics)
}

func (r *taskResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	lifecycle := r.t.Lifecycle()
	if lifecycle.Teardown == "" {
		resp.Diagnostics.AddWarning(
			"Resource removed from state only",
			fmt.Sprintf("%s has no teardown, so destroying it leaves the server unchanged.", r.t.Key),
		)
		return
	}
	values, _, err := toValues(r.t.Fields(), req.State.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Invalid state", err.Error())
		return
	}
	task, err := buildTask(r.t, values, lifecycle.Apply)
	if err != nil {
		resp.Diagnostics.AddError("Invalid state", err.Error())
		return
	}
	teardown, err := sdk.TaskInState(task, lifecycle.Teardown)
	if err != nil {
		resp.Diagnostics.AddError("Unable to destroy the resource", err.Error())
		return
	}

	ctx = r.data.context(ctx)
	out := teardown.Execute(ctx)
	entries := sdk.Diagnostics(ctx, out)
	if out.Error != nil || hasErrors(entries) {
		// A resource whose app or service is already gone is destroyed
		// with it.
		if _, _, err := sdk.ReadTask(ctx, task); errors.Is(err, sdk.ErrObjectMissing) {
			return
		}
	}
	addDiagnostics(&resp.Diagnostics, entries, false)
}

func (r *taskResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	typeKey, _, err := sdk.ParseIdentityAddress(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import address", err.Error())
		return
	}
	if typeKey != r.t.Key {
		resp.Diagnostics.AddError(
			"Invalid import address",
			fmt.Sprintf("the address names a %s, not a %s", typeKey, r.t.Key),
		)
		return
	}
	if r.t.Schema.Imperative {
		resp.Diagnostics.AddError(
			"Import not supported",
			fmt.Sprintf("%s runs an operation rather than declaring a resource, so there is nothing to import.", r.t.Key),
		)
		return
	}

	ctx = r.data.context(ctx)
	read, exists, err := sdk.ReadAddress(ctx, req.ID)
	if err != nil {
		addDiagnostics(&resp.Diagnostics, sdk.Diagnostics(ctx, err), false)
		return
	}
	if !exists {
		resp.Diagnostics.AddError("Resource not found", fmt.Sprintf("%s does not exist on the server.", req.ID))
		return
	}
	typ := objectType(ctx, resp.State.Schema)
	r.store(ctx, read, observedValues(read), tftypes.NewValue(typ, nil), &resp.State, &resp.Diagnostics)
}
