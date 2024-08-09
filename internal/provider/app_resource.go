package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &appResource{}
	_ resource.ResourceWithConfigure   = &appResource{}
	_ resource.ResourceWithImportState = &appResource{}
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
	BLE    types.Bool `tfsdk:"ble"`
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

type ScopeListValidator struct{}

func (s ScopeListValidator) Description(ctx context.Context) string {
	return "Validate LIFF scope. openid, profile, chat_message.write are available."
}
func (s ScopeListValidator) MarkdownDescription(ctx context.Context) string {
	return "Validate LIFF scope. `openid`, `profile`, `chat_message.write` are available."
}
func (s ScopeListValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	elements := req.ConfigValue.Elements()
	if len(elements) == 0 {
		resp.Diagnostics.AddAttributeError(req.Path, "Empty scope", "At least one of openid or profile is required.")
	}

	foundOpenid := false
	foundProfile := false
	for _, element := range elements {
		item := element.String()
		if item != "\"openid\"" && item != "\"profile\"" && item != "\"chat_message.write\"" {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid scope",
				fmt.Sprintf("Invalid scope %s. `openid`, `profile`, `chat_message.write` are available.", item),
			)
		}
		if item == "\"openid\"" {
			foundOpenid = true
		}
		if item == "\"profile\"" {
			foundProfile = true
		}
	}
	if !foundOpenid && !foundProfile {
		resp.Diagnostics.AddAttributeError(req.Path, "Empty scope", "At least one of `openid` or `profile` is required.")
	}
}

type ScopeListPlanSortingModifier struct{}

func (s ScopeListPlanSortingModifier) Description(ctx context.Context) string {
	return "Ensure that the scope list is sorted."
}
func (s ScopeListPlanSortingModifier) MarkdownDescription(ctx context.Context) string {
	return "Ensure that the scope list is sorted."
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				Required: true,
				Attributes: map[string]schema.Attribute{
					"ble": schema.BoolAttribute{
						Description: "If BLE is available.",
						Computed:    true,
					},
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
				Required:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.UniqueValues(),
					listvalidator.IsRequired(),
					listvalidator.SizeAtLeast(1),
					listvalidator.ValueStringsAre(
						stringvalidator.OneOf("openid", "profile", "chat_message.write"),
					),
					ScopeListValidator{},
				},
				PlanModifiers: []planmodifier.List{},
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
		if !plan.Features.QRCode.IsNull() {
			qrCode := plan.Features.QRCode.ValueBool()
			appCreateRequest.Features.QRCode = &qrCode
		}
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
	liffApp, err := r.client.GetLiffApp(createdLiffId)

	if err != nil {
		resp.Diagnostics.AddError("Failed to list Get apps", err.Error())
		return
	}

	plan.LiffId = types.StringValue(liffApp.LiffId)
	plan.View = &appResourceViewModel{
		Type:       types.StringValue(liffApp.View.Type),
		URL:        types.StringValue(liffApp.View.URL),
		ModuleMode: types.BoolValue(*liffApp.View.ModuleMode),
	}
	plan.Features = &appResourceFeaturesModel{}

	plan.Features.BLE = types.BoolValue(liffApp.Features.BLE)
	plan.Features.QRCode = types.BoolValue(liffApp.Features.QRCode)

	plan.Description = types.StringValue(*liffApp.Description)

	plan.PermanentLinkPattern = types.StringValue(liffApp.PermanentLinkPattern)

	plan.BotPrompt = types.StringValue(liffApp.BotPrompt)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *appResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state appResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	liffApp, err := r.client.GetLiffApp(state.LiffId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Failed to Get LIFF apps", err.Error())
		return
	}

	state.LiffId = types.StringValue(liffApp.LiffId)
	state.View = &appResourceViewModel{
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

	state.Features = &appResourceFeaturesModel{
		BLE:    types.BoolValue(liffApp.Features.BLE),
		QRCode: types.BoolValue(liffApp.Features.QRCode),
	}

	if liffApp.Scope != nil {
		state.Scope = []types.String{}
		for _, scope := range liffApp.Scope {
			state.Scope = append(state.Scope, types.StringValue(scope))
		}
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *appResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan appResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state appResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var updateRequest LiffAppUpdateRequest
	if plan.View != nil {
		updateRequest.View = LiffAppUpdateRequestView{}
		if !plan.View.Type.IsNull() && plan.View.Type.ValueString() != "" {
			typeValue := plan.View.Type.ValueString()
			updateRequest.View.Type = &typeValue
		}
		if !plan.View.URL.IsNull() && plan.View.URL.ValueString() != "" {
			urlValue := plan.View.URL.ValueString()
			updateRequest.View.URL = &urlValue
		}
		moduleMode := plan.View.ModuleMode.ValueBool()
		updateRequest.View.ModuleMode = &moduleMode
	}

	if !plan.Description.IsNull() && plan.Description.ValueString() != "" {
		descriptionValue := plan.Description.ValueString()
		updateRequest.Description = &descriptionValue
	}

	if plan.Features != nil {
		updateRequest.Features = &LiffAppUpdateRequestFeatures{}
		if !plan.Features.QRCode.IsNull() {
			qrCode := plan.Features.QRCode.ValueBool()
			updateRequest.Features.QRCode = &qrCode
		}
	}

	if !plan.PermanentLinkPattern.IsNull() && plan.PermanentLinkPattern.ValueString() != "" {
		permanentLinkPattern := plan.PermanentLinkPattern.ValueString()
		updateRequest.PermanentLinkPattern = &permanentLinkPattern
	}

	if plan.Scope != nil {
		updateRequest.Scope = &[]string{}
		for _, scope := range plan.Scope {
			*updateRequest.Scope = append(*updateRequest.Scope, scope.ValueString())
		}
	}

	if !plan.BotPrompt.IsNull() && plan.BotPrompt.ValueString() != "" {
		botPrompt := plan.BotPrompt.ValueString()
		updateRequest.BotPrompt = &botPrompt
	}

	updatedLiffId := state.LiffId.ValueString()

	tflog.Debug(ctx, "Updating LIFF app with LINE API Client")
	updateError := r.client.UpdateLiffApp(updatedLiffId, updateRequest)

	if updateError != nil {
		resp.Diagnostics.AddError("Failed to update LIFF app", updateError.Error())
		return
	}

	// obtain again
	liffApp, err := r.client.GetLiffApp(updatedLiffId)

	if err != nil {
		resp.Diagnostics.AddError("Failed to list Get apps", err.Error())
		return
	}

	plan.LiffId = types.StringValue(liffApp.LiffId)
	plan.View = &appResourceViewModel{
		Type:       types.StringValue(liffApp.View.Type),
		URL:        types.StringValue(liffApp.View.URL),
		ModuleMode: types.BoolValue(*liffApp.View.ModuleMode),
	}
	plan.Features = &appResourceFeaturesModel{}

	plan.Features.BLE = types.BoolValue(liffApp.Features.BLE)
	plan.Features.QRCode = types.BoolValue(liffApp.Features.QRCode)

	plan.Description = types.StringValue(*liffApp.Description)

	plan.PermanentLinkPattern = types.StringValue(liffApp.PermanentLinkPattern)

	plan.BotPrompt = types.StringValue(liffApp.BotPrompt)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *appResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	var state appResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteLiffApp(state.LiffId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Failed to delete LIFF app", err.Error())
		return
	}
}

func (r *appResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	liffId := req.ID
	liffApp, err := r.client.GetLiffApp(liffId)

	if err != nil {
		resp.Diagnostics.AddError("Failed to Get LIFF apps", err.Error())
		return
	}

	var state appResourceModel
	state.LiffId = types.StringValue(liffApp.LiffId)
	state.View = &appResourceViewModel{
		Type: types.StringValue(liffApp.View.Type),
		URL:  types.StringValue(liffApp.View.URL),
	}
	state.BotPrompt = types.StringValue(liffApp.BotPrompt)
	if liffApp.View.ModuleMode != nil {
		state.View.ModuleMode = types.BoolValue(*liffApp.View.ModuleMode)
	}
	if liffApp.Description != nil {
		state.Description = types.StringValue(*liffApp.Description)
	}
	state.PermanentLinkPattern = types.StringValue(liffApp.PermanentLinkPattern)

	state.Features = &appResourceFeaturesModel{
		BLE:    types.BoolValue(liffApp.Features.BLE),
		QRCode: types.BoolValue(liffApp.Features.QRCode),
	}

	if liffApp.Scope != nil {
		state.Scope = []types.String{}
		for _, scope := range liffApp.Scope {
			state.Scope = append(state.Scope, types.StringValue(scope))
		}
	}

	diags := resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
