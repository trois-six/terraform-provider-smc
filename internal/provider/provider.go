// Copyright (c) HashiCorp, Inc.

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/trois-six/smc"
)

// Ensure SMCProvider satisfies various provider interfaces.
var _ provider.Provider = &SMCProvider{}

// SMCProvider defines the provider implementation.
type SMCProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// SMCProviderModel describes the provider data model.
type SMCProviderModel struct {
	Hostname types.String `tfsdk:"hostname"`
	APIKey   types.String `tfsdk:"api_key"`
}

func (p *SMCProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "smc"
	resp.Version = p.version
}

func (p *SMCProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with the SMC Management API",
		Attributes: map[string]schema.Attribute{
			"hostname": schema.StringAttribute{
				MarkdownDescription: "URI for the SMC Management API. May also be provided via SMC_HOSTNAME environment variable.",
				Optional:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "API Key for the SMC Management API. May also be provided via SMC_API_KEY environment variable.",
				Sensitive:           true,
				Optional:            true,
			},
		},
	}
}

func (p *SMCProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring SMC client")

	var data SMCProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Hostname.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("hostname"),
			"Unknown SMC hostname",
			"The provider cannot create the SMC client as there is an unknown configuration value for the SMC hostname. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the SMC_HOSTNAME environment variable.",
		)
	}

	if data.APIKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Unknown API Key hostname",
			"The provider cannot create the SMC client as there is an unknown configuration value for the SMC API Key. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the SMC_API_KEY environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	hostname := os.Getenv("SMC_HOSTNAME")
	apiKey := os.Getenv("SMC_API_KEY")

	if !data.Hostname.IsNull() {
		hostname = data.Hostname.ValueString()
	}

	if !data.APIKey.IsNull() {
		apiKey = data.APIKey.ValueString()
	}

	if hostname == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("hostname"),
			"Missing SMC hostname",
			"The provider cannot create the SMC client as there is a missing or empty value for the SMC hostname. "+
				"Set the hostname value in the configuration or use the SMC_HOSTNAME environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if data.APIKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing SMC API Key",
			"The provider cannot create the SMC client as there is a missing or empty value for the SMC API Key. "+
				"Set the hostname value in the configuration or use the SMC_API_KEY environment variable. "+
				"If either is already set, ensure the value is not empty.")
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "smc_hostname", hostname)
	ctx = tflog.SetField(ctx, "smc_api_key", apiKey)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "smc_api_key")

	tflog.Debug(ctx, "Creating SMC client")

	// Create a new SMC client using the configuration values
	client, err := smc.NewSMCClientWithResponses(hostname, apiKey)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create SMC Client",
			"An unexpected error occurred when creating the SMC client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"SMC Client Error: "+err.Error(),
		)
		return
	}

	// Make the SMC client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client

	tflog.Info(ctx, "Configured SMC client", map[string]any{"success": true})
}

func (p *SMCProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}

func (p *SMCProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAccountDataSource,
		NewAccountsDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &SMCProvider{
			version: version,
		}
	}
}
