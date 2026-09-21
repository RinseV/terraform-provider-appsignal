// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/RinseV/appsignal-client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &appNotifiersDataSource{}
	_ datasource.DataSourceWithConfigure = &appNotifiersDataSource{}
)

// NewAppNotifiersDataSource is a helper function to simplify the provider implementation.
func NewAppNotifiersDataSource() datasource.DataSource {
	return &appNotifiersDataSource{}
}

// appNotifiersDataSource is the data source implementation.
type appNotifiersDataSource struct {
	client *appsignal.Client
}

type appNotifiersDataSourceModel struct {
	AppID     types.String                 `tfsdk:"app_id"`
	Name      types.String                 `tfsdk:"name"`
	Notifiers []appNotifierDataSourceModel `tfsdk:"notifiers"`
}

type appNotifierDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Icon types.String `tfsdk:"icon"`
}

// Configure adds the provider configured client to the data source.
func (d *appNotifiersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*appsignalProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *appsignalProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = providerData.client
}

// Metadata returns the data source type name.
func (d *appNotifiersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_notifiers"
}

// Schema defines the schema for the data source.
func (d *appNotifiersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to look up the notifiers of an existing app, optionally filtered by name.",
		Attributes: map[string]schema.Attribute{
			"app_id": schema.StringAttribute{
				Description: "The ID of the app. Must be specified.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name to filter the notifiers by. Only notifiers with exactly this name are returned. When omitted, all notifiers of the app are returned.",
				Optional:    true,
			},
			"notifiers": schema.ListNestedAttribute{
				Description: "The notifiers matching the given filters.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the notifier.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the notifier.",
							Computed:    true,
						},
						"icon": schema.StringAttribute{
							Description: "The icon of the notifier.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *appNotifiersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state appNotifiersDataSourceModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	notifiers, err := d.client.GetAppNotifiers(ctx, state.AppID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read AppSignal App Notifiers",
			err.Error(),
		)
		return
	}

	state.Notifiers = []appNotifierDataSourceModel{}
	for _, notifier := range notifiers {
		if !state.Name.IsNull() && notifier.Name != state.Name.ValueString() {
			continue
		}

		state.Notifiers = append(state.Notifiers, appNotifierDataSourceModel{
			ID:   types.StringValue(notifier.ID),
			Name: types.StringValue(notifier.Name),
			Icon: types.StringValue(notifier.Icon),
		})
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
