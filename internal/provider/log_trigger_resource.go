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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &logTriggerResource{}
	_ resource.ResourceWithConfigure   = &logTriggerResource{}
	_ resource.ResourceWithImportState = &logTriggerResource{}
)

func NewLogTriggerResource() resource.Resource {
	return &logTriggerResource{}
}

// logTriggerResource is the resource implementation
type logTriggerResource struct {
	client       *appsignal.Client
	organization string
}

var logTriggerActionTypes = []string{
	string(appsignal.ActionTypeTrigger),
	string(appsignal.ActionTypeFilter),
	string(appsignal.ActionTypeMetrics),
}

var logTriggerSeverities = []string{
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

var logTriggerNotificationOptions = []string{
	string(appsignal.NotificationOptionAlways),
	string(appsignal.NotificationOptionNever),
	string(appsignal.NotificationOptionFirstInDeploy),
	string(appsignal.NotificationOptionFirstAfterClose),
	string(appsignal.NotificationOptionNthInHour),
	string(appsignal.NotificationOptionNthInDay),
}

type logTriggerResourceModel struct {
	ID                       types.String `tfsdk:"id"`
	AppID                    types.String `tfsdk:"app_id"`
	Name                     types.String `tfsdk:"name"`
	Query                    types.String `tfsdk:"query"`
	SourceIDs                types.Set    `tfsdk:"source_ids"`
	ActionType               types.String `tfsdk:"action_type"`
	Description              types.String `tfsdk:"description"`
	NotificationOptions      types.String `tfsdk:"notification_options"`
	NotificationTriggerValue types.Int32  `tfsdk:"notification_trigger_value"`
	Order                    types.Int32  `tfsdk:"order"`
	Severities               types.Set    `tfsdk:"severities"`
	NotifierIDs              types.Set    `tfsdk:"notifier_ids"`
}

// applyLogTrigger copies the values AppSignal owns into the model
func (m *logTriggerResourceModel) applyLogTrigger(logTrigger *appsignal.LogTrigger) {
	m.ID = types.StringValue(logTrigger.ID)
	m.Name = types.StringValue(logTrigger.Name)
	m.Query = types.StringValue(logTrigger.Query)
	m.ActionType = types.StringValue(string(logTrigger.ActionType))
	m.Order = types.Int32Value(logTrigger.Order)
}

// Configure adds the provider configured client to the resource.
func (r *logTriggerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *logTriggerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_log_trigger"
}

// Schema defines the schema for the resource.
func (r *logTriggerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the log trigger.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"app_id": schema.StringAttribute{
				Description: "The ID of the app to which the log trigger belongs.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the log trigger.",
				Required:    true,
			},
			"query": schema.StringAttribute{
				Description: "The query string that will execute the log trigger.",
				Required:    true,
			},
			"action_type": schema.StringAttribute{
				MarkdownDescription: "The action the log trigger performs. Assigned by AppSignal, one of: `" +
					strings.Join(logTriggerActionTypes, "`, `") + "`.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the log trigger. Omit the attribute to clear it.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"notification_options": schema.StringAttribute{
				MarkdownDescription: "Notification option. One of: `" +
					strings.Join(logTriggerNotificationOptions, "`, `") + "`.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(string(appsignal.NotificationOptionAlways)),
				Validators: []validator.String{
					stringvalidator.OneOf(logTriggerNotificationOptions...),
				},
			},
			"notification_trigger_value": schema.Int32Attribute{
				Description: "Notification threshold.",
				Optional:    true,
				Computed:    true,
				Default:     int32default.StaticInt32(1),
			},
			"severities": schema.SetAttribute{
				MarkdownDescription: "The severities the log trigger matches. Each one of: `" +
					strings.Join(logTriggerSeverities, "`, `") + "`.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(stringvalidator.OneOf(logTriggerSeverities...)),
				},
			},
			"source_ids": schema.SetAttribute{
				Description: "The IDs of the log sources for which this trigger is active.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
			},
			"notifier_ids": schema.SetAttribute{
				Description: "The IDs of the notifiers this trigger notifies.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
			},
			"order": schema.Int32Attribute{
				Description: "The order of the log trigger.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *logTriggerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan logTriggerResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var severities []appsignal.LogSeverity
	var sourceIDs, notifierIDs []string
	resp.Diagnostics.Append(plan.Severities.ElementsAs(ctx, &severities, true)...)
	resp.Diagnostics.Append(plan.SourceIDs.ElementsAs(ctx, &sourceIDs, true)...)
	resp.Diagnostics.Append(plan.NotifierIDs.ElementsAs(ctx, &notifierIDs, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var input appsignal.CreateAppLogTriggerInput
	input.AppID = plan.AppID.ValueString()
	input.Name = plan.Name.ValueString()
	input.Query = plan.Query.ValueString()
	input.Description = plan.Description.ValueStringPointer()
	input.NotificationOptions = logTriggerNotificationOptionPointer(plan.NotificationOptions)
	input.NotificationTriggerValue = plan.NotificationTriggerValue.ValueInt32Pointer()
	input.Severities = severities
	input.SourceIDs = sourceIDs
	input.NotifierIDs = notifierIDs

	logTrigger, err := r.client.CreateAppLogTrigger(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating log trigger",
			"Could not create log trigger, unexpected error: "+err.Error(),
		)
		return
	}

	plan.applyLogTrigger(logTrigger)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *logTriggerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state logTriggerResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	logTriggers, err := r.client.GetAppLogTriggers(ctx, state.AppID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading AppSignal Log Trigger",
			"Could not read AppSignal log trigger ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	var logTrigger *appsignal.LogTrigger
	for i, candidate := range logTriggers {
		if candidate.ID == state.ID.ValueString() {
			logTrigger = &logTriggers[i]
			break
		}
	}

	if logTrigger == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	severities, severitiesDiags := types.SetValueFrom(ctx, types.StringType, logTrigger.Severities)
	resp.Diagnostics.Append(severitiesDiags...)
	sourceIDs, sourceIDsDiags := types.SetValueFrom(ctx, types.StringType, logTrigger.SourceIDs)
	resp.Diagnostics.Append(sourceIDsDiags...)

	ids := make([]string, 0, len(logTrigger.Notifiers))
	for _, notifier := range logTrigger.Notifiers {
		ids = append(ids, notifier.ID)
	}
	notifierIDs, notifierIDsDiags := types.SetValueFrom(ctx, types.StringType, ids)
	resp.Diagnostics.Append(notifierIDsDiags...)

	if resp.Diagnostics.HasError() {
		return
	}

	state.applyLogTrigger(logTrigger)

	// A cleared description comes back empty; configuration expresses that by
	// leaving the attribute out.
	state.Description = types.StringNull()
	if logTrigger.Description != "" {
		state.Description = types.StringValue(logTrigger.Description)
	}
	state.NotificationOptions = types.StringValue(string(logTrigger.NotificationOptions))
	state.NotificationTriggerValue = types.Int32Value(logTrigger.NotificationTriggerValue)
	state.Severities = severities
	state.SourceIDs = sourceIDs
	state.NotifierIDs = notifierIDs

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *logTriggerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan logTriggerResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var severities []appsignal.LogSeverity
	var sourceIDs, notifierIDs []string
	resp.Diagnostics.Append(plan.Severities.ElementsAs(ctx, &severities, true)...)
	resp.Diagnostics.Append(plan.SourceIDs.ElementsAs(ctx, &sourceIDs, true)...)
	resp.Diagnostics.Append(plan.NotifierIDs.ElementsAs(ctx, &notifierIDs, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The API ignores a null description, so clearing one means sending an
	// empty string instead of leaving the variable out of the mutation.
	description := plan.Description.ValueStringPointer()
	if description == nil {
		description = new(string)
	}

	var input appsignal.UpdateAppLogTriggerInput
	input.AppID = plan.AppID.ValueString()
	input.LogTriggerID = plan.ID.ValueString()
	input.Name = plan.Name.ValueStringPointer()
	input.Query = plan.Query.ValueStringPointer()
	input.Description = description
	input.NotificationOptions = logTriggerNotificationOptionPointer(plan.NotificationOptions)
	input.NotificationTriggerValue = plan.NotificationTriggerValue.ValueInt32Pointer()
	input.Severities = severities
	input.SourceIDs = sourceIDs
	input.NotifierIDs = notifierIDs

	logTrigger, err := r.client.UpdateAppLogTrigger(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating AppSignal Log Trigger",
			"Could not update log trigger, unexpected error "+err.Error(),
		)
		return
	}

	plan.applyLogTrigger(logTrigger)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *logTriggerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state logTriggerResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var input appsignal.DeleteAppLogTriggerInput
	input.AppID = state.AppID.ValueString()
	input.LogTriggerID = state.ID.ValueString()

	_, err := r.client.DeleteAppLogTrigger(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting AppSignal Log Trigger",
			"Could not delete log trigger, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *logTriggerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// A log trigger can only be read through the app that owns it, so the import
	// ID has to carry both: "<app id>,<log trigger id>".
	appID, logTriggerID, found := strings.Cut(req.ID, ",")
	if !found || appID == "" || logTriggerID == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected an import identifier of the form \"app_id,log_trigger_id\", got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_id"), appID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), logTriggerID)...)
}

// logTriggerNotificationOptionPointer converts a configured notification option
// into the enum pointer the API expects, leaving it unset when absent.
func logTriggerNotificationOptionPointer(value types.String) *appsignal.LogTriggerNotificationOption {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	option := appsignal.LogTriggerNotificationOption(value.ValueString())

	return &option
}
