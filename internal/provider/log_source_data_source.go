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
	_ datasource.DataSource              = &logSourceDataSource{}
	_ datasource.DataSourceWithConfigure = &logSourceDataSource{}
)

// NewLogSourceDataSource is a helper function to simplify the provider implementation.
func NewLogSourceDataSource() datasource.DataSource {
	return &logSourceDataSource{}
}

// logSourceDataSource is the data source implementation.
type logSourceDataSource struct {
	client       *appsignal.Client
	organization string
}

type logSourceDataSourceModel struct {
	ID    types.String `tfsdk:"id"`
	AppID types.String `tfsdk:"app_id"`
	Name  types.String `tfsdk:"name"`
	Key   types.String `tfsdk:"key"`
	Type  types.String `tfsdk:"type"`
	Fmt   types.String `tfsdk:"fmt"`
}

// Configure adds the provider configured client to the data source.
func (d *logSourceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *logSourceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_log_source"
}

// Schema defines the schema for the data source.
func (d *logSourceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to look up an existing log source of an app by name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the log source.",
				Computed:    true,
			},
			"app_id": schema.StringAttribute{
				Description: "The ID of the app the log source belongs to. Must be specified.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the log source. Log source names are unique within an app.",
				Required:    true,
			},
			"key": schema.StringAttribute{
				Description: "The API key used for the log source.",
				Computed:    true,
				Sensitive:   true,
			},
			"type": schema.StringAttribute{
				Description: "The type of the log source.",
				Computed:    true,
			},
			"fmt": schema.StringAttribute{
				Description: "The format of the log source.",
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *logSourceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state logSourceDataSourceModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The API can only list all of an app's log sources, so the lookup by name
	// happens here.
	logSources, err := d.client.GetAppLogSources(ctx, state.AppID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read AppSignal App Log Sources",
			err.Error(),
		)
		return
	}

	var found *appsignal.LogSource
	for _, logSource := range logSources {
		if logSource.Name == state.Name.ValueString() {
			found = &logSource
			break
		}
	}

	if found == nil {
		resp.Diagnostics.AddError(
			"AppSignal Log Source Not Found",
			fmt.Sprintf("No log source named %q exists in app %q.", state.Name.ValueString(), state.AppID.ValueString()),
		)
		return
	}

	state.ID = types.StringValue(found.ID)
	state.Name = types.StringValue(found.Name)
	state.Key = types.StringValue(found.Key)
	state.Type = types.StringValue(found.Type)
	state.Fmt = types.StringValue(string(found.Fmt))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
