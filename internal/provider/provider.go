// Package provider implements the Dokku Terraform provider on the docket SDK.
// Every resource and data source is generated from docket's task catalog.
package provider

import (
	"context"
	"os"
	"strconv"

	"github.com/dokku/docket/sdk"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ provider.Provider                   = (*dokkuProvider)(nil)
	_ provider.ProviderWithValidateConfig = (*dokkuProvider)(nil)
)

// dokkuProvider is the provider. runner is set only by tests, to run every
// command against a fake dokku.
type dokkuProvider struct {
	version string
	runner  sdk.ExecRunner
}

type providerModel struct {
	Host              types.String `tfsdk:"host"`
	Sudo              types.Bool   `tfsdk:"sudo"`
	AcceptNewHostKeys types.Bool   `tfsdk:"accept_new_host_keys"`
	SSHIdentityFile   types.String `tfsdk:"ssh_identity_file"`
	SSHOptions        types.List   `tfsdk:"ssh_options"`
	IgnoreVersion     types.Bool   `tfsdk:"ignore_version"`
}

// New returns the provider constructor for a release version.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &dokkuProvider{version: version}
	}
}

func (p *dokkuProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "dokku"
	resp.Version = p.version
}

func (p *dokkuProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Dokku server through the docket SDK.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The server to manage, as `[user@]host[:port]`, reached over SSH. Empty runs `dokku` on the machine Terraform runs on. Defaults to `DOKKU_HOST`.",
			},
			"sudo": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Run `dokku` through `sudo`. Defaults to `DOKKU_SUDO`.",
			},
			"accept_new_host_keys": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Trust a host key SSH has not seen before. A changed key is still refused. Defaults to `DOKKU_ACCEPT_NEW_HOST_KEYS`.",
			},
			"ssh_identity_file": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The private key SSH offers, alone. Without it, SSH picks a key from ssh-agent or `~/.ssh/config`. Defaults to `DOKKU_SSH_IDENTITY_FILE`.",
			},
			"ssh_options": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Extra SSH options, each one `-o` value such as `ConnectTimeout=10`. `ControlMaster`, `ControlPath`, `ControlPersist` and `BatchMode` are refused.",
			},
			"ignore_version": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Run tasks even when Dokku, or a plugin a task needs, is older than the task supports. Defaults to `DOKKU_IGNORE_VERSION`, else false.",
			},
		},
	}
}

func (p *dokkuProvider) ValidateConfig(ctx context.Context, req provider.ValidateConfigRequest, resp *provider.ValidateConfigResponse) {
	var model providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() || model.SSHOptions.IsNull() || model.SSHOptions.IsUnknown() {
		return
	}
	var opts []types.String
	resp.Diagnostics.Append(model.SSHOptions.ElementsAs(ctx, &opts, false)...)
	known := make([]string, 0, len(opts))
	for _, o := range opts {
		if !o.IsUnknown() && !o.IsNull() {
			known = append(known, o.ValueString())
		}
	}
	if err := sdk.ValidateSSHOptions(known); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("ssh_options"), "Invalid SSH option", err.Error())
	}
}

func (p *dokkuProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var model providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	target := sdk.Target{
		Host:              stringSetting(model.Host, "DOKKU_HOST"),
		Sudo:              boolSetting(model.Sudo, "DOKKU_SUDO"),
		AcceptNewHostKeys: boolSetting(model.AcceptNewHostKeys, "DOKKU_ACCEPT_NEW_HOST_KEYS"),
		IdentityFile:      stringSetting(model.SSHIdentityFile, "DOKKU_SSH_IDENTITY_FILE"),
	}
	if !model.SSHOptions.IsNull() {
		resp.Diagnostics.Append(model.SSHOptions.ElementsAs(ctx, &target.SSHOptions, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	data := &providerData{
		session: newSession(!boolSetting(model.IgnoreVersion, "DOKKU_IGNORE_VERSION")),
		target:  target,
		runner:  p.runner,
	}
	resp.ResourceData = data
	resp.DataSourceData = data
}

// newSession builds the session every call shares. The provider process is
// killed without a shutdown hook, so no call can close the session; SSH
// connections close as soon as no command is using them instead.
func newSession(versionGate bool) *sdk.Session {
	opts := []sdk.SessionOption{sdk.WithControlPersist(0)}
	if versionGate {
		opts = append(opts, sdk.WithVersionGate())
	}
	return sdk.NewSession(sdk.NewMasker(), opts...)
}

// stringSetting is the configured value, else the environment variable.
func stringSetting(v types.String, env string) string {
	if !v.IsNull() && !v.IsUnknown() {
		return v.ValueString()
	}
	return os.Getenv(env)
}

// boolSetting is the configured value, else the environment variable parsed
// as a bool, else false.
func boolSetting(v types.Bool, env string) bool {
	if !v.IsNull() && !v.IsUnknown() {
		return v.ValueBool()
	}
	b, err := strconv.ParseBool(os.Getenv(env))
	return err == nil && b
}

func (p *dokkuProvider) Resources(_ context.Context) []func() resource.Resource {
	types, err := taskTypes()
	if err != nil {
		panic(err)
	}
	out := make([]func() resource.Resource, 0, len(types))
	for _, t := range types {
		out = append(out, newTaskResource(t))
	}
	return out
}

func (p *dokkuProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	types, err := taskTypes()
	if err != nil {
		panic(err)
	}
	out := []func() datasource.DataSource{newExportDataSource}
	for _, t := range types {
		if t.Readable() {
			out = append(out, newTaskDataSource(t))
		}
	}
	return out
}
