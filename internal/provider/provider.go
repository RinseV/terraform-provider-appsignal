// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	appsignal "github.com/RinseV/appsignal-client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &appsignalProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &appsignalProvider{
			version: version,
		}
	}
}

// appsignalProvider is the provider implementation.
type appsignalProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// appsignalProviderModel maps provider schema data to a Go type.
type appsignalProviderModel struct {
	Host             types.String `tfsdk:"host"`
	Token            types.String `tfsdk:"token"`
	OrganizationSlug types.String `tfsdk:"organization_slug"`
}

type appsignalProviderData struct {
	client           *appsignal.Client
	organizationSlug string
}

// Metadata returns the provider type name.
func (p *appsignalProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "appsignal"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *appsignalProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "URI for AppSignal API. May also be provided via APPSIGNAL_HOST environment variable.",
				Optional:    true,
			},
			"token": schema.StringAttribute{
				Description: "Token for AppSignal API. May also be provided via APPSIGNAL_TOKEN environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"organization_slug": schema.StringAttribute{
				Description: "Slug of the organization to manage. Every data source and resource works in this organization. May also be provided via APPSIGNAL_ORGANIZATION_SLUG environment variable.",
				Optional:    true,
			},
		},
	}
}

func (p *appsignalProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring AppSignal client")

	// Retrieve provider data from configuration
	var config appsignalProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if config.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown AppSignal API Host",
			"The provider cannot create the AppSignal API client as there is an unknown configuration value for the AppSignal API host. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the APPSIGNAL_HOST environment variable.",
		)
	}

	if config.OrganizationSlug.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("organization_slug"),
			"Unknown AppSignal Organization Slug",
			"The provider cannot create the AppSignal API client as there is an unknown configuration value for the AppSignal organization slug. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the APPSIGNAL_ORGANIZATION_SLUG environment variable.",
		)
	}

	if config.Token.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Unknown AppSignal API Token",
			"The provider cannot create the AppSignal API client as there is an unknown configuration value for the AppSignal API token. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the APPSIGNAL_TOKEN environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	host := os.Getenv("APPSIGNAL_HOST")
	token := os.Getenv("APPSIGNAL_TOKEN")
	organizationSlug := os.Getenv("APPSIGNAL_ORGANIZATION_SLUG")

	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}

	if !config.Token.IsNull() {
		token = config.Token.ValueString()
	}

	if !config.OrganizationSlug.IsNull() {
		organizationSlug = config.OrganizationSlug.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Missing AppSignal API Host",
			"The provider cannot create the AppSignal API client as there is a missing or empty value for the AppSignal API host. "+
				"Set the host value in the configuration or use the APPSIGNAL_HOST environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Missing AppSignal API Token",
			"The provider cannot create the AppSignal API client as there is a missing or empty value for the AppSignal API token. "+
				"Set the token value in the configuration or use the APPSIGNAL_TOKEN environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if organizationSlug == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("organization_slug"),
			"Missing AppSignal Organization Slug",
			"The provider cannot manage AppSignal resources as there is a missing or empty value for the AppSignal organization slug. "+
				"Set the organization_slug value in the configuration or use the APPSIGNAL_ORGANIZATION_SLUG environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "appsignal_host", host)
	ctx = tflog.SetField(ctx, "appsignal_token", token)
	ctx = tflog.SetField(ctx, "appsignal_organization_slug", organizationSlug)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "appsignal_token")

	tflog.Debug(ctx, "Creating AppSignal client")

	// Create a new AppSignal client using the configuration values
	client := appsignal.NewClient(host, token)

	// Make the AppSignal client and the default organization slug available
	// during DataSource and Resource type Configure methods.
	providerData := &appsignalProviderData{
		client:           client,
		organizationSlug: organizationSlug,
	}

	resp.DataSourceData = providerData
	resp.ResourceData = providerData

	tflog.Info(ctx, "Configured AppSignal client", map[string]any{"success": true})
}

// DataSources defines the data sources implemented in the provider.
func (p *appsignalProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewViewerDataSource,
		NewAppDataSource,
		NewOrganizationDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *appsignalProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAppResource,
		NewLogSourceResource,
		NewLogTriggerResource,
	}
}
