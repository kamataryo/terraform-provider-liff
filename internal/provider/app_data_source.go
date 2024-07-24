package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &appDataSource{}
	_ datasource.DataSourceWithConfigure = &appDataSource{}
)

func NewAppDataSource() datasource.DataSource {
	return &appDataSource{}
}

type appDataSource struct {
	client *LineApiClient
}

type appDataSourceViewModel struct {
	Type       types.String `tfsdk:"type"`
	URL        types.String `tfsdk:"url"`
	ModuleMode types.Bool   `tfsdk:"module_mode"`
}
type appDataSourceFeaturesModel struct {
	BLE    types.Bool `tfsdk:"ble"`
	QRCode types.Bool `tfsdk:"qr_code"`
}

type appDataSourceModel struct {
	LiffId               types.String                `tfsdk:"liff_id"`
	View                 *appDataSourceViewModel     `tfsdk:"view"`
	Description          types.String                `tfsdk:"description"`
	PermanentLinkPattern types.String                `tfsdk:"permanent_link_pattern"`
	Features             *appDataSourceFeaturesModel `tfsdk:"features"`
	Scope                []types.String              `tfsdk:"scope"`
	BotPrompt            types.String                `tfsdk:"bot_prompt"`
}

func (d *appDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*LineApiClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *LineApiClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *appDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

func (d *appDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"liff_id": schema.StringAttribute{
				Description: "The LIFF app ID",
				Required:    true,
			},
			"view": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "Screen size of the LIFF app. full, tall or compact are available",
						Computed:    true,
					},
					"url": schema.StringAttribute{
						Description: "The endpoint URL of the LIFF app",
						Computed:    true,
					},
					"module_mode": schema.BoolAttribute{
						Description: "if the LIFF app is in module mode",
						Computed:    true,
						Optional:    true,
					},
				},
			},
			"description": schema.StringAttribute{
				Description: "Name of LIFF app",
				Computed:    true,
			},
			"permanent_link_pattern": schema.StringAttribute{
				Description: "How to add LIFF URL. value concat will be return.",
				Computed:    true,
			},
			"features": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"ble": schema.BoolAttribute{
						Description: "If Bluetooth Low Energy (BLE) is available with LINE Things service.",
						Computed:    true,
					},
					"qr_code": schema.BoolAttribute{
						Description: "If QR code reader is available within the LIFF app.",
						Computed:    true,
					},
				},
			},
			"scope": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"bot_prompt": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *appDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	var data appDataSourceModel
	req.Config.Get(ctx, &data)

	statelessChannelAccessToken, err := d.client.GetStatelessChannelAccessTokenV3()
	if err != nil {
		resp.Diagnostics.AddError("Failed to get stateless channel access token", err.Error())
		return
	}

	liffApps, err := d.client.ListLiffApps(statelessChannelAccessToken)

	if err != nil {
		resp.Diagnostics.AddError("Failed to list LIFF apps", err.Error())
		return
	}

	var target LiffApp

	for _, liffApp := range liffApps {
		if liffApp.LiffId == data.LiffId.ValueString() {
			target = liffApp
			break
		}
	}

	var output appDataSourceModel
	output.LiffId = types.StringValue(target.LiffId)
	output.View = &appDataSourceViewModel{
		Type:       types.StringValue(target.View.Type),
		URL:        types.StringValue(target.View.URL),
		ModuleMode: types.BoolValue(target.View.ModuleMode),
	}
	output.Description = types.StringValue(target.Description)
	output.PermanentLinkPattern = types.StringValue(target.PermanentLinkPattern)

	if target.Features != nil {
		output.Features = &appDataSourceFeaturesModel{
			BLE:    types.BoolValue(target.Features.BLE),
			QRCode: types.BoolValue(target.Features.QRCode),
		}
	}

	output.Scope = []types.String{}
	for _, scope := range target.Scope {
		output.Scope = append(output.Scope, types.StringValue(scope))
	}
	output.BotPrompt = types.StringValue(target.BotPrompt)

	diags := resp.State.Set(ctx, &output)
	resp.Diagnostics.Append(diags...)
}
