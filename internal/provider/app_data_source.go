package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/RinseV/appsignal-client-go"
	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource                     = &appDataSource{}
	_ datasource.DataSourceWithConfigure        = &appDataSource{}
	_ datasource.DataSourceWithConfigValidators = &appDataSource{}
)

// NewAppDataSource is a helper function to simplify the provider implementation.
func NewAppDataSource() datasource.DataSource {
	return &appDataSource{}
}

// appDataSource is the data source implementation.
type appDataSource struct {
	client       *appsignal.Client
	organization string
}

type appDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Environment types.String `tfsdk:"environment"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

// Configure adds the provider configured client to the data source.
func (d *appDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *appDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

// Schema defines the schema for the data source.
func (d *appDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to look up an existing app, either by ID or by its name and environment combination.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the app. Either this or the combination of name and environment must be specified.",
				Optional:    true,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the app. Must be specified together with environment, and cannot be combined with id.",
				Optional:    true,
				Computed:    true,
			},
			"environment": schema.StringAttribute{
				Description: "The environment of the app. Must be specified together with name, and cannot be combined with id.",
				Optional:    true,
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
	}
}

// ConfigValidators ensures the app is looked up either by ID or by the
// combination of name and environment, but never by a mix of the two.
func (d *appDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("name"),
		),
		datasourcevalidator.RequiredTogether(
			path.MatchRoot("name"),
			path.MatchRoot("environment"),
		),
		datasourcevalidator.Conflicting(
			path.MatchRoot("id"),
			path.MatchRoot("environment"),
		),
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *appDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config appDataSourceModel

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var app *appsignal.App

	if !config.ID.IsNull() {
		app = d.readByID(ctx, config.ID.ValueString(), &resp.Diagnostics)
	} else {
		app = d.readByNameAndEnvironment(ctx, config.Name.ValueString(), config.Environment.ValueString(), &resp.Diagnostics)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	state := appDataSourceModel{
		ID:          types.StringValue(app.ID),
		Name:        types.StringValue(app.Name),
		Environment: types.StringValue(app.Environment),
		CreatedAt:   types.StringValue(app.CreatedAt.Format(time.RFC3339)),
		UpdatedAt:   types.StringValue(app.UpdatedAt.Format(time.RFC3339)),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// readByID looks up a single app by its ID.
func (d *appDataSource) readByID(ctx context.Context, id string, diags *diag.Diagnostics) *appsignal.App {
	app, err := d.client.GetApp(ctx, id)
	if err != nil {
		diags.AddError(
			"Unable to Read AppSignal App",
			err.Error(),
		)
		return nil
	}

	if app == nil {
		diags.AddError(
			"AppSignal App Not Found",
			fmt.Sprintf("No app with ID %q was found.", id),
		)
		return nil
	}

	return app
}

// readByNameAndEnvironment looks up a single app within the configured
// organization by its name and environment combination, which is unique per
// organization.
func (d *appDataSource) readByNameAndEnvironment(ctx context.Context, name string, environment string, diags *diag.Diagnostics) *appsignal.App {
	apps, err := d.client.GetOrganizationApps(ctx, d.organization)
	if err != nil {
		diags.AddError(
			"Unable to Read AppSignal Organization Apps",
			err.Error(),
		)
		return nil
	}

	for _, app := range apps {
		if app.Name == name && app.Environment == environment {
			return &app
		}
	}

	diags.AddError(
		"AppSignal App Not Found",
		fmt.Sprintf("No app with name %q and environment %q was found in organization %q.", name, environment, d.organization),
	)
	return nil
}
