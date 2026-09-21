package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/RinseV/appsignal-client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &appsDataSource{}
	_ datasource.DataSourceWithConfigure = &appsDataSource{}
)

// NewAppsDataSource is a helper function to simplify the provider implementation.
func NewAppsDataSource() datasource.DataSource {
	return &appsDataSource{}
}

// appsDataSource is the data source implementation.
type appsDataSource struct {
	client *appsignal.Client

	// organization is the slug of the organization to list the apps of,
	// configured on the provider.
	organization string
}

type appsDataSourceModel struct {
	Organization types.String                              `tfsdk:"organization"`
	Apps         map[string]organizationAppDataSourceModel `tfsdk:"apps"`
}

type organizationAppDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Environment types.String `tfsdk:"environment"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

// Configure adds the provider configured client to the data source.
func (d *appsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	d.organization = providerData.organization
}

// Metadata returns the data source type name.
func (d *appsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_apps"
}

// Schema defines the schema for the data source.
func (d *appsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to look up every app of the organization configured on the provider.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description: "The slug of the organization the apps were listed from, as configured on the provider.",
				Computed:    true,
			},
			"apps": schema.MapNestedAttribute{
				Description: "Every app of the organization, keyed by app ID.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the app.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the app.",
							Computed:    true,
						},
						"environment": schema.StringAttribute{
							Description: "The environment of the app.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "The timestamp when the app was created.",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "The timestamp when the app was updated.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *appsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state appsDataSourceModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apps, err := d.client.GetOrganizationApps(ctx, d.organization)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read AppSignal Organization Apps",
			err.Error(),
		)
		return
	}

	state.Organization = types.StringValue(d.organization)

	// Keying by ID rather than returning a list keeps the map stable when the
	// API changes the order it returns the apps in, so a reorder does not show
	// up as a diff in everything that iterates over them.
	state.Apps = map[string]organizationAppDataSourceModel{}
	for _, app := range apps {
		state.Apps[app.ID] = organizationAppDataSourceModel{
			ID:          types.StringValue(app.ID),
			Name:        types.StringValue(app.Name),
			Environment: types.StringValue(app.Environment),
			CreatedAt:   types.StringValue(app.CreatedAt.Format(time.RFC3339)),
			UpdatedAt:   types.StringValue(app.UpdatedAt.Format(time.RFC3339)),
		}
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
