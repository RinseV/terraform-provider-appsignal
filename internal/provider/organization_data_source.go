package provider

import (
	"context"
	"fmt"

	"github.com/RinseV/appsignal-client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &organizationDataSource{}
	_ datasource.DataSourceWithConfigure = &organizationDataSource{}
)

func NewOrganizationDataSource() datasource.DataSource { return &organizationDataSource{} }

type organizationDataSource struct {
	client *appsignal.Client

	// organization is the slug of the organization to look up, configured on
	// the provider.
	organization string
}

type organizationDataSourceModel struct {
	Slug types.String `tfsdk:"slug"`
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

// Configure adds the provider configured client to the data source.
func (d *organizationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *organizationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

// Schema defines the schema for the data source.
func (d *organizationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to look up the organization configured on the provider.",
		Attributes: map[string]schema.Attribute{
			"slug": schema.StringAttribute{
				Description: "The slug of the organization, as configured on the provider.",
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Description: "The organization id.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the organization.",
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *organizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state organizationDataSourceModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organization, err := d.client.GetOrganization(ctx, d.organization)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read AppSignal Organization",
			err.Error(),
		)
		return
	}

	state = organizationDataSourceModel{
		Slug: types.StringValue(d.organization),
		ID:   types.StringValue(organization.ID),
		Name: types.StringValue(organization.Name),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
