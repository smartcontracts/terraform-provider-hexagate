package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces
var _ provider.Provider = &HexagateProvider{}

// HexagateProvider is the provider implementation.
type HexagateProvider struct {
	// version is set to the provider version on release.
	version string
}

// ProviderClient wraps the HexagateClient with additional provider-specific data
type Client struct {
	HexagateClient *HexagateClient
	UserAgent      string
}

// HexagateProviderModel describes the provider data model.
type HexagateProviderModel struct {
	APIToken types.String `tfsdk:"api_token"`
	APIURL   types.String `tfsdk:"api_url"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &HexagateProvider{
			version: version,
		}
	}
}

// Metadata returns the provider type name.
func (p *HexagateProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "hexagate"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *HexagateProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with Hexagate.",
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "The API token for Hexagate API authentication.",
			},
			"api_url": schema.StringAttribute{
				Optional:    true,
				Description: "The URL for the Hexagate API.",
			},
		},
	}
}

// Configure prepares a Hexagate API client for data sources and resources.
func (p *HexagateProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config HexagateProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Default values
	apiURL := "https://api.hexagate.com/api/v2"
	if !config.APIURL.IsNull() {
		apiURL = config.APIURL.ValueString()
	}

	if config.APIToken.IsNull() {
		resp.Diagnostics.AddError(
			"Missing API Token Configuration",
			"While configuring the provider, the API token was not found. "+
				"Please configure the api_token attribute in the provider configuration block.",
		)
		return
	}

	// Create a custom User-Agent for API requests
	userAgent := fmt.Sprintf("terraform-provider-hexagate/%s", p.version)

	client := &Client{
		HexagateClient: &HexagateClient{
			APIToken: config.APIToken.ValueString(),
			BaseURL:  apiURL,
			Client:   &http.Client{},
		},
		UserAgent: userAgent,
	}

	// Test the API connection
	_, err := client.HexagateClient.GetAllMonitors()
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Connect to Hexagate API",
			fmt.Sprintf("Failed to connect to API: %v", err),
		)
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

// DataSources defines the data sources implemented in the provider.
func (p *HexagateProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// We'll implement these later
		// NewMonitorDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *HexagateProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewMonitorResource,
	}
}
