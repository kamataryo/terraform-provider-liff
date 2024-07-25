package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &appResource{}
	_ resource.ResourceWithConfigure = &appResource{}
)

// appResource is the resource implementation.
type appResource struct {
	client *LineApiClient
}

type appResourceViewModel struct {
	Type       types.String `tfsdk:"type"`
	URL        types.String `tfsdk:"url"`
	ModuleMode types.Bool   `tfsdk:"module_mode"`
}

type appResourceFeaturesModel struct {
	QRCode types.Bool `tfsdk:"qr_code"`
}

type appResourceModel struct {
	LiffId               types.String              `tfsdk:"liff_id"`
	View                 *appResourceViewModel     `tfsdk:"view"`
	Description          types.String              `tfsdk:"description"`
	Features             *appResourceFeaturesModel `tfsdk:"features"`
	PermanentLinkPattern types.String              `tfsdk:"permanent_link_pattern"`
	Scope                []types.String            `tfsdk:"scope"`
	BotPrompt            types.String              `tfsdk:"bot_prompt"`
}

// NewAppResource is a helper function to simplify the provider implementation.
func NewAppResource() resource.Resource {
	return &appResource{}
}

// Metadata returns the resource type name.
func (r *appResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

func (r *appResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*LineApiClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *LineApiClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// Schema defines the schema for the resource.
func (r *appResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {

	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"liff_id": schema.StringAttribute{
				Description: "The LIFF ID.",
				Computed:    true,
			},
			"view": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "Screen size of the LIFF app. full, tall or compact are available",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOfCaseInsensitive([]string{"full", "tall", "compact"}...),
						},
					},
					"url": schema.StringAttribute{
						Description: "The endpoint URL of the LIFF app",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.RegexMatches(
								regexp.MustCompile(`^https://.*$`),
								"URL must be HTTPS",
							),
						},
					},
					"module_mode": schema.BoolAttribute{
						Description: "If the LIFF app is in module mode",
						Optional:    true,
						Computed:    true,
					},
				},
			},
			"description": schema.StringAttribute{
				Description: "Name of LIFF app",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 256),
				},
			},
			"features": schema.SingleNestedAttribute{
				// Computed: true,
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"qr_code": schema.BoolAttribute{
						Description: "If QR code is available.",
						Optional:    true,
						Computed:    true,
					},
				},
			},
			"permanent_link_pattern": schema.StringAttribute{
				Description: "How to add LIFF URL. Specify concat.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("concat"),
				},
			},
			"scope": schema.ListAttribute{
				Description: "The permission of the LIFF app.",
				Optional:    true,
				// Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.UniqueValues(),
					listvalidator.ValueStringsAre(
						stringvalidator.OneOf("openid", "email", "profile", "chat_message.write"),
					),
				},
			},
			"bot_prompt": schema.StringAttribute{
				Description: "Add friends options",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("normal", "aggressive", "none"),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *appResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan appResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	var appCreateRequest LiffAppCreateRequest
	if plan.View != nil {
		appCreateRequest.View = LiffAppCreateRequestView{
			Type: plan.View.Type.ValueString(),
			URL:  plan.View.URL.ValueString(),
		}
		// if plan.View.ModuleMode != nil {
		moduleMode := plan.View.ModuleMode.ValueBool()
		appCreateRequest.View.ModuleMode = &moduleMode
	}

	description := plan.Description.ValueString()
	appCreateRequest.Description = &description

	if plan.Features != nil {
		appCreateRequest.Features = &LiffAppCreateRequestFeatures{}
		// if plan.Features.QRCode != nil {
		qrCode := plan.Features.QRCode.ValueBool()
		appCreateRequest.Features.QRCode = &qrCode
		// }
	}

	if !plan.PermanentLinkPattern.IsNull() && plan.PermanentLinkPattern.ValueString() != "" {
		permanentLinkPattern := plan.PermanentLinkPattern.ValueString()
		appCreateRequest.PermanentLinkPattern = &permanentLinkPattern
	}

	if plan.Scope != nil {
		appCreateRequest.Scope = &[]string{}
		for _, scope := range plan.Scope {
			*appCreateRequest.Scope = append(*appCreateRequest.Scope, scope.ValueString())
		}
	}

	if !plan.BotPrompt.IsNull() && plan.BotPrompt.ValueString() != "" {
		botPrompt := plan.BotPrompt.ValueString()
		appCreateRequest.BotPrompt = &botPrompt
	}

	tflog.Debug(ctx, "Creating LIFF app with LINE API Client")
	createdLiffId, err := r.client.CreateLiffApp(appCreateRequest)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create LIFF app", err.Error())
		return
	}

	// obtain again
	liffApps, err := r.client.ListLiffApps()

	if err != nil {
		resp.Diagnostics.AddError("Failed to list LIFF apps", err.Error())
		return
	}

	target := LiffAppsListResponseItem{}
	found := false
	for _, liffApp := range liffApps {
		if liffApp.LiffId == createdLiffId {
			target = liffApp
			found = true
			break
		}
	}

	println("target:")
	println(fmt.Sprintf("%+v", target))

	if !found {
		resp.Diagnostics.AddError("Failed to find created LIFF app", "Failed to find created LIFF app")
		return
	}

	output := appResourceModel{}

	output.LiffId = types.StringValue(target.LiffId)
	output.View = &appResourceViewModel{
		Type:       types.StringValue(target.View.Type),
		URL:        types.StringValue(target.View.URL),
		ModuleMode: types.BoolValue(*target.View.ModuleMode),
	}

	if target.Description != nil {
		output.Description = types.StringValue(*target.Description)
	}

	output.PermanentLinkPattern = types.StringValue(target.PermanentLinkPattern)

	diags = resp.State.Set(ctx, &output)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *appResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data appResourceModel
	req.State.Get(ctx, &data)

	tflog.Info(ctx, "Reading LIFF apps with LINE API Client")
	liffApps, err := r.client.ListLiffApps()

	if err != nil {
		resp.Diagnostics.AddError("Failed to list LIFF apps", err.Error())
		return
	}

	var target LiffAppsListResponseItem

	for _, liffApp := range liffApps {
		if liffApp.LiffId == data.LiffId.ValueString() {
			target = liffApp
			break
		}
	}

	var output appDataSourceModel
	output.LiffId = types.StringValue(target.LiffId)
	output.View = &appDataSourceViewModel{
		Type: types.StringValue(target.View.Type),
		URL:  types.StringValue(target.View.URL),
	}
	if target.View.ModuleMode != nil {
		output.View.ModuleMode = types.BoolValue(*target.View.ModuleMode)
	}
	if target.Description != nil {
		output.Description = types.StringValue(*target.Description)
	}
	output.PermanentLinkPattern = types.StringValue(target.PermanentLinkPattern)

	if target.Features != nil {
		output.Features = &appDataSourceFeaturesModel{
			BLE:    types.BoolValue(target.Features.BLE),
			QRCode: types.BoolValue(target.Features.QRCode),
		}
	}

	if target.Scope != nil {
		output.Scope = []types.String{}
		for _, scope := range target.Scope {
			output.Scope = append(output.Scope, types.StringValue(scope))
		}
	}
	output.BotPrompt = types.StringValue(target.BotPrompt)

	diags := resp.State.Set(ctx, &output)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *appResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *appResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}
