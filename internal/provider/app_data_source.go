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

	var state appDataSourceModel
	req.Config.Get(ctx, &state)

	liffApp, err := d.client.GetLiffApp(state.LiffId.ValueString())
	if liffApp == nil || err != nil {
		resp.Diagnostics.AddError("Failed to Get LIFF apps", err.Error())
		return
	}

	state.View = &appDataSourceViewModel{
		Type: types.StringValue(liffApp.View.Type),
		URL:  types.StringValue(liffApp.View.URL),
	}
	if liffApp.View.ModuleMode != nil {
		state.View.ModuleMode = types.BoolValue(*liffApp.View.ModuleMode)
	}
	if liffApp.Description != nil {
		state.Description = types.StringValue(*liffApp.Description)
	}
	state.PermanentLinkPattern = types.StringValue(liffApp.PermanentLinkPattern)

	if liffApp.Features != nil {
		state.Features = &appDataSourceFeaturesModel{
			BLE:    types.BoolValue(liffApp.Features.BLE),
			QRCode: types.BoolValue(liffApp.Features.QRCode),
		}
	}

	if liffApp.Scope != nil {
		state.Scope = []types.String{}
		for _, scope := range liffApp.Scope {
			state.Scope = append(state.Scope, types.StringValue(scope))
		}
	}
	state.BotPrompt = types.StringValue(liffApp.BotPrompt)

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
