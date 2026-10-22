package provider

import (
	"context"
	"fmt"

	"github.com/dokku/docket/sdk"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var _ datasource.DataSourceWithConfigure = (*taskDataSource)(nil)

// taskDataSource reads one resource of a docket task type by its identity
// keys.
type taskDataSource struct {
	t    *taskType
	data *providerData
}

// newTaskDataSource returns the data source constructor for a task type.
func newTaskDataSource(t *taskType) func() datasource.DataSource {
	return func() datasource.DataSource {
		return &taskDataSource{t: t}
	}
}

func (d *taskDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = d.t.Key
}

func (d *taskDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	s, err := dataSourceSchema(d.t)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported task schema", err.Error())
		return
	}
	resp.Schema = s
}

func (d *taskDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*providerData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("expected *providerData, got %T", req.ProviderData))
		return
	}
	d.data = data
}

// keyFields are the identity keys, the data source's inputs.
func (d *taskDataSource) keyFields() []sdk.FieldSchema {
	var keys []sdk.FieldSchema
	for _, f := range d.t.Fields() {
		if f.Identity == sdk.IdentityRoleKey {
			keys = append(keys, f)
		}
	}
	return keys
}

func (d *taskDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	values, _, err := toValues(d.keyFields(), req.Config.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Invalid configuration", err.Error())
		return
	}
	task, err := buildTask(d.t, values, d.t.Lifecycle().Apply)
	if err != nil {
		resp.Diagnostics.AddError("Invalid configuration", err.Error())
		return
	}
	address := sdk.IdentityAddress(d.t.Key, task)

	ctx = d.data.context(ctx)
	read, exists, err := sdk.ReadAddress(ctx, address)
	if err != nil {
		addDiagnostics(&resp.Diagnostics, sdk.Diagnostics(ctx, err), false)
		return
	}
	if !exists {
		resp.Diagnostics.AddError("Resource not found", fmt.Sprintf("%s does not exist on the server.", address))
		return
	}

	typ := objectType(ctx, resp.State.Schema)
	id := tftypes.NewValue(tftypes.String, address)
	raw, err := fromObserved(d.t.Fields(), typ, observedValues(read), req.Config.Raw, map[string]tftypes.Value{idAttribute: id})
	if err != nil {
		resp.Diagnostics.AddError("Unable to record the data source state", err.Error())
		return
	}
	resp.State.Raw = raw
}
