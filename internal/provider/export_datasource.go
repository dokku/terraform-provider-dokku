package provider

import (
	"context"
	"fmt"

	"github.com/dokku/docket/sdk"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSourceWithConfigure = (*exportDataSource)(nil)

// exportDataSource reads a whole server, or the apps or resources it names,
// back as a docket recipe.
type exportDataSource struct {
	data *providerData
}

type exportModel struct {
	Apps             types.List   `tfsdk:"apps"`
	Resources        types.List   `tfsdk:"resources"`
	Format           types.String `tfsdk:"format"`
	Redact           types.Bool   `tfsdk:"redact"`
	ID               types.String `tfsdk:"id"`
	Recipe           types.String `tfsdk:"recipe"`
	Vars             types.Map    `tfsdk:"vars"`
	Warnings         types.List   `tfsdk:"warnings"`
	MissingApps      types.List   `tfsdk:"missing_apps"`
	MissingResources types.List   `tfsdk:"missing_resources"`
}

func newExportDataSource() datasource.DataSource {
	return &exportDataSource{}
}

func (d *exportDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_export"
}

func (d *exportDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the server, or the apps or resources it names, back as a docket recipe.",
		Attributes: map[string]schema.Attribute{
			"apps": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Only export these apps. The global play is skipped. Conflicts with `resources`.",
				Validators:          []validator.List{listvalidator.ConflictsWith(path.MatchRoot("resources"))},
			},
			"resources": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Only export these resource addresses, such as `dokku_config[app=api]`.",
			},
			"format": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The recipe format: `yaml` (the default), `json5` or `hcl`.",
				Validators:          []validator.String{stringvalidator.OneOf(sdk.FormatYAML, sdk.FormatJSON5, sdk.FormatHCL)},
			},
			"redact": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Replace sensitive values with placeholders. Defaults to true. When false, sensitive values are lifted into `vars`.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The format of the export.",
			},
			"recipe": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The exported recipe.",
			},
			"vars": schema.MapAttribute{
				Computed:            true,
				Sensitive:           true,
				ElementType:         types.StringType,
				MarkdownDescription: "The values the recipe refers to as inputs, keyed by input name.",
			},
			"warnings": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Non-fatal problems the export ran into.",
			},
			"missing_apps": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Apps named in `apps` that do not exist.",
			},
			"missing_resources": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Addresses named in `resources` that matched nothing.",
			},
		},
	}
}

func (d *exportDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *exportDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model exportModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := sdk.ExportOptions{Redact: model.Redact.IsNull() || model.Redact.ValueBool()}
	resp.Diagnostics.Append(model.Apps.ElementsAs(ctx, &opts.Apps, false)...)
	var addresses []string
	resp.Diagnostics.Append(model.Resources.ElementsAs(ctx, &addresses, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(addresses) > 0 {
		selectors, err := sdk.ParseResourceSelectors(addresses)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("resources"), "Invalid resource address", err.Error())
			return
		}
		opts.Resources = selectors
	}
	format := sdk.FormatYAML
	if !model.Format.IsNull() {
		format = model.Format.ValueString()
	}

	ctx = d.data.context(ctx)
	res, err := sdk.ExportRecipe(ctx, opts)
	if err != nil {
		addDiagnostics(&resp.Diagnostics, sdk.Diagnostics(ctx, err), false)
		return
	}
	recipe, err := res.MarshalRecipe(format)
	if err != nil {
		resp.Diagnostics.AddError("Unable to render the recipe", err.Error())
		return
	}

	// Warnings may quote values the export read, so they are masked with
	// what it found sensitive.
	masker := sdk.NewMasker(res.SensitiveValues()...)
	warnings := make([]string, len(res.Report.Warnings))
	for i, w := range res.Report.Warnings {
		warnings[i] = masker.String(w)
	}

	diags := &resp.Diagnostics
	model.ID = types.StringValue(format)
	model.Recipe = types.StringValue(string(recipe))
	model.Vars = mustMap(ctx, res.Vars, diags)
	model.Warnings = mustList(ctx, warnings, diags)
	model.MissingApps = mustList(ctx, res.Report.MissingApps, diags)
	model.MissingResources = mustList(ctx, res.Report.MissingResources, diags)
	if diags.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

// mustList converts a string slice into a list value, never null.
func mustList(ctx context.Context, values []string, diags *diag.Diagnostics) types.List {
	if values == nil {
		values = []string{}
	}
	list, d := types.ListValueFrom(ctx, types.StringType, values)
	diags.Append(d...)
	return list
}

// mustMap converts a string map into a map value, never null.
func mustMap(ctx context.Context, values map[string]string, diags *diag.Diagnostics) types.Map {
	if values == nil {
		values = map[string]string{}
	}
	m, d := types.MapValueFrom(ctx, types.StringType, values)
	diags.Append(d...)
	return m
}
