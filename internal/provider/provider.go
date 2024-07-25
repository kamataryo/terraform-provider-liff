package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &liffProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &liffProvider{
			version: version,
		}
	}
}

// liffProvider is the provider implementation.
type liffProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type liffProviderModel struct {
	ChannelId     types.String `tfsdk:"channel_id"`
	ChannelSecret types.String `tfsdk:"channel_secret"`
}

// Metadata returns the provider type name.
func (p *liffProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "liff"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *liffProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"channel_id": schema.StringAttribute{
				Optional: true,
			},
			"channel_secret": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
		},
	}
}

// Configure prepares a HashiCups API client for data sources and resources.
func (p *liffProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring LINE Messaging API client")

	var config liffProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	var channel_id string = ""
	var channel_secret string = ""

	if config.ChannelId.IsNull() {
		channel_id = os.Getenv("LINE_CHANNEL_ID")
	} else {
		channel_id = config.ChannelId.ValueString()
	}

	if config.ChannelSecret.IsNull() {
		channel_secret = os.Getenv("LINE_CHANNEL_SECRET")
	} else {
		channel_secret = config.ChannelSecret.ValueString()
	}

	if channel_id == "" {
		resp.Diagnostics.AddError(
			"channel_id is required",
			"channel_id is required",
		)
	}

	if channel_secret == "" {
		resp.Diagnostics.AddError(
			"channel_secret is required",
			"channel_secret is required",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client, err := LineMessagingAPIClient(channel_id, channel_secret)

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create Line Messaging API client",
			"Failed to create Line Messaging API client: "+err.Error(),
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client

	tflog.Info(ctx, "Configured LINE Messaging API client", map[string]any{"success": true})
}

// DataSources defines the data sources implemented in the provider.
func (p *liffProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAppDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *liffProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAppResource,
	}
}
