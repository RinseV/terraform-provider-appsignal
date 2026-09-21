// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/RinseV/appsignal-client-go"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &logViewResource{}
	_ resource.ResourceWithConfigure   = &logViewResource{}
	_ resource.ResourceWithImportState = &logViewResource{}
)

func NewLogViewResource() resource.Resource { return &logViewResource{} }

// logViewResource is the resource implementation.
type logViewResource struct {
	client       *appsignal.Client
	organization string
}

const logViewDefaultLineHeight = "0"

var logViewSeverities = []string{
	string(appsignal.SeverityTrace),
	string(appsignal.SeverityDebug),
	string(appsignal.SeverityInfo),
	string(appsignal.SeverityNotice),
	string(appsignal.SeverityWarn),
	string(appsignal.SeverityError),
	string(appsignal.SeverityCritical),
	string(appsignal.SeverityAlert),
	string(appsignal.SeverityFatal),
	string(appsignal.SeverityUnknown),
}

type logViewResourceModel struct {
	ID         types.String `tfsdk:"id"`
	AppID      types.String `tfsdk:"app_id"`
	Name       types.String `tfsdk:"name"`
	Query      types.String `tfsdk:"query"`
	Columns    types.Set    `tfsdk:"columns"`
	LineHeight types.String `tfsdk:"line_height"`
	Severities types.Set    `tfsdk:"severities"`
	SourceIDs  types.Set    `tfsdk:"source_ids"`
}

// applyLogView copies the values AppSignal owns into the model.
func (m *logViewResourceModel) applyLogView(logView *appsignal.LogView) {
	m.ID = types.StringValue(logView.ID)
	m.Name = types.StringValue(logView.Name)
}

// Configure adds the provider configured client to the resource.
func (r *logViewResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.organization = providerData.organization
}

// Metadata returns the resource type name.
func (r *logViewResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_log_view"
}

// Schema defines the schema for the resource.
func (r *logViewResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the log view.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"app_id": schema.StringAttribute{
				Description: "The ID of the app to which the log view belongs.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the log view.",
				Required:    true,
			},
			"query": schema.StringAttribute{
				Description: "The query string that is used for the log view. Defaults to an empty string.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"columns": schema.SetAttribute{
				Description: "The columns to include in the log view.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
			},
			"line_height": schema.StringAttribute{
				MarkdownDescription: "The line height of the log view. Defaults to `" +
					logViewDefaultLineHeight + "`, the value AppSignal itself assigns.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(logViewDefaultLineHeight),
			},
			"severities": schema.SetAttribute{
				MarkdownDescription: "The severities the log view matches. Each one of: `" +
					strings.Join(logViewSeverities, "`, `") + "`.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(stringvalidator.OneOf(logViewSeverities...)),
				},
			},
			"source_ids": schema.SetAttribute{
				Description: "The IDs of the log sources which this view aggregates.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *logViewResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan logViewResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var severities []appsignal.LogSeverity
	var sourceIDs, columns []string
	resp.Diagnostics.Append(plan.Severities.ElementsAs(ctx, &severities, true)...)
	resp.Diagnostics.Append(plan.SourceIDs.ElementsAs(ctx, &sourceIDs, true)...)
	resp.Diagnostics.Append(plan.Columns.ElementsAs(ctx, &columns, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var input appsignal.CreateAppLogViewInput
	input.AppID = plan.AppID.ValueString()
	input.Name = plan.Name.ValueString()
	input.Query = plan.Query.ValueStringPointer()
	input.LineHeight = plan.LineHeight.ValueStringPointer()
	input.Severities = severities
	input.SourceIDs = sourceIDs
	input.Columns = columns

	logView, err := r.client.CreateAppLogView(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating log view",
			"Could not create log view, unexpected error: "+err.Error(),
		)
		return
	}

	plan.applyLogView(logView)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *logViewResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state logViewResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	logView, err := r.client.GetAppLogView(ctx, state.AppID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading AppSignal Log View",
			"Could not read AppSignal log view ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Gone from AppSignal, so Terraform plans it as a create again.
	if logView == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	columns, columnsDiags := types.SetValueFrom(ctx, types.StringType, logView.Columns)
	resp.Diagnostics.Append(columnsDiags...)
	sourceIDs, sourceIDsDiags := types.SetValueFrom(ctx, types.StringType, logView.SourceIDs)
	resp.Diagnostics.Append(sourceIDsDiags...)
	severities, severitiesDiags := types.SetValueFrom(ctx, types.StringType, logView.Severities)
	resp.Diagnostics.Append(severitiesDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.applyLogView(logView)
	state.Query = types.StringValue(logView.Query)
	state.LineHeight = types.StringValue(logView.LineHeight)
	state.Columns = columns
	state.SourceIDs = sourceIDs
	state.Severities = severities

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *logViewResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan logViewResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var severities []appsignal.LogSeverity
	var sourceIDs, columns []string
	resp.Diagnostics.Append(plan.Severities.ElementsAs(ctx, &severities, true)...)
	resp.Diagnostics.Append(plan.SourceIDs.ElementsAs(ctx, &sourceIDs, true)...)
	resp.Diagnostics.Append(plan.Columns.ElementsAs(ctx, &columns, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var input appsignal.UpdateAppLogViewInput
	input.AppID = plan.AppID.ValueString()
	input.LogViewID = plan.ID.ValueString()
	input.Name = plan.Name.ValueStringPointer()
	input.Query = plan.Query.ValueStringPointer()
	input.LineHeight = plan.LineHeight.ValueStringPointer()
	input.Severities = severities
	input.SourceIDs = sourceIDs
	input.Columns = columns

	logView, err := r.client.UpdateAppLogView(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating AppSignal Log View",
			"Could not update log view, unexpected error "+err.Error(),
		)
		return
	}

	plan.applyLogView(logView)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *logViewResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state logViewResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var input appsignal.DeleteAppLogViewInput
	input.AppID = state.AppID.ValueString()
	input.LogViewID = state.ID.ValueString()

	_, err := r.client.DeleteAppLogView(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting AppSignal Log View",
			"Could not delete log view, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *logViewResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// A log view can only be read through the app that owns it, so the import
	// ID has to carry both: "<app id>,<log view id>".
	appID, logViewID, found := strings.Cut(req.ID, ",")
	if !found || appID == "" || logViewID == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected an import identifier of the form \"app_id,log_view_id\", got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_id"), appID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), logViewID)...)
}
