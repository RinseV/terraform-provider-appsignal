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
	_ datasource.DataSource              = &viewerDataSource{}
	_ datasource.DataSourceWithConfigure = &viewerDataSource{}
)

// NewViewerDataSource is a helper function to simplify the provider implementation.
func NewViewerDataSource() datasource.DataSource {
	return &viewerDataSource{}
}

// viewerDataSource is the data source implementation.
type viewerDataSource struct {
	client *appsignal.Client
}

type viewerDataSourceModel struct {
	ID    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Email types.String `tfsdk:"email"`
}

// Configure adds the provider configured client to the data source.
func (d *viewerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*appsignal.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *appsignal.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

// Metadata returns the data source type name.
func (d *viewerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_viewer"
}

// Schema defines the schema for the data source.
func (d *viewerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the viewer.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the viewer.",
				Computed:    true,
			},
			"email": schema.StringAttribute{
				Description: "The email of the viewer.",
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *viewerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state viewerDataSourceModel

	viewer, err := d.client.GetViewer(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read AppSignal Viewer",
			err.Error(),
		)
		return
	}

	state = viewerDataSourceModel{
		ID:    types.StringValue(viewer.ID),
		Name:  types.StringValue(viewer.Name),
		Email: types.StringValue(viewer.Email),
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
