// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/RinseV/appsignal-client-go"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &logSourceResource{}
	_ resource.ResourceWithConfigure   = &logSourceResource{}
	_ resource.ResourceWithImportState = &logSourceResource{}
)

// NewLogSourceResource is a helper function to simplify the provider implementation.
func NewLogSourceResource() resource.Resource {
	return &logSourceResource{}
}

// logSourceResource is the resource implementation.
type logSourceResource struct {
	client           *appsignal.Client
	organizationSlug string
}

var logSourceFormats = []string{
	string(appsignal.LogSourceFormatPlaintext),
	string(appsignal.LogSourceFormatLogfmt),
	string(appsignal.LogSourceFormatJSON),
	string(appsignal.LogSourceFormatAutodetect),
}

type logSourceResourceModel struct {
	ID    types.String `tfsdk:"id"`
	AppID types.String `tfsdk:"app_id"`
	Name  types.String `tfsdk:"name"`
	Key   types.String `tfsdk:"key"`
	Type  types.String `tfsdk:"type"`
	Fmt   types.String `tfsdk:"fmt"`
}

// Configure adds the provider configured client to the resource.
func (r *logSourceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*appsignalProviderData)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *appsignalProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = providerData.client
	r.organizationSlug = providerData.organizationSlug
}

// Metadata returns the resource type name.
func (r *logSourceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_log_source"
}

// Schema defines the schema for the resource.
func (r *logSourceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the log source.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"app_id": schema.StringAttribute{
				Description: "The ID of the app to which the log source belongs.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the log source.",
				Required:    true,
			},
			"key": schema.StringAttribute{
				Description: "The API key used for the log source.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"fmt": schema.StringAttribute{
				MarkdownDescription: "The format of the log source. One of: `" +
					strings.Join(logSourceFormats, "`, `") + "`.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf(logSourceFormats...),
				},
			},
			"type": schema.StringAttribute{
				Description: "The type of the log source.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *logSourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan logSourceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var input appsignal.CreateAppLogSourceInput
	input.AppID = plan.AppID.ValueString()
	input.Name = plan.Name.ValueString()
	input.Type = plan.Type.ValueString()
	input.Fmt = appsignal.LogSourceFormat(plan.Fmt.ValueString())

	logSource, err := r.client.CreateAppLogSource(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating log source",
			"Could not create log source, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(logSource.ID)
	plan.Name = types.StringValue(logSource.Name)
	plan.Key = types.StringValue(logSource.Key)
	plan.Type = types.StringValue(logSource.Type)
	plan.Fmt = types.StringValue(string(logSource.Fmt))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *logSourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state logSourceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	logSource, err := r.client.GetAppLogSource(ctx, state.AppID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading AppSignal Log Source",
			"Could not read AppSignal log source ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	state.Name = types.StringValue(logSource.Name)
	state.Key = types.StringValue(logSource.Key)
	state.Type = types.StringValue(logSource.Type)
	state.Fmt = types.StringValue(string(logSource.Fmt))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *logSourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan logSourceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var input appsignal.UpdateAppLogSourceInput
	input.LogSourceID = plan.ID.ValueString()
	input.AppID = plan.AppID.ValueString()
	input.Name = plan.Name.ValueString()
	input.Fmt = appsignal.LogSourceFormat(plan.Fmt.ValueString())

	logSource, err := r.client.UpdateAppLogSource(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating AppSignal Log Source",
			"Could not update log source, unexpected error "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(logSource.ID)
	plan.Name = types.StringValue(logSource.Name)
	plan.Key = types.StringValue(logSource.Key)
	plan.Type = types.StringValue(logSource.Type)
	plan.Fmt = types.StringValue(string(logSource.Fmt))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *logSourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state logSourceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var input appsignal.DeleteAppLogSourceInput
	input.AppID = state.AppID.ValueString()
	input.LogSourceID = state.ID.ValueString()

	_, err := r.client.DeleteAppLogSource(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting AppSignal Log Source",
			"Could not delete log source, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *logSourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// A log source can only be read through the app that owns it, so the import
	// ID has to carry both: "<app id>,<log source id>".
	appID, logSourceID, found := strings.Cut(req.ID, ",")
	if !found || appID == "" || logSourceID == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected an import identifier of the form \"app_id,log_source_id\", got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_id"), appID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), logSourceID)...)
}
