package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func init() {
	schema.DescriptionKind = schema.StringMarkdown
}

func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"api_token": {
					Type:        schema.TypeString,
					Required:    true,
					Sensitive:   true,
					Description: "The API token for Hexagate API authentication",
				},
				"api_url": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "https://api.hexagate.com/api/v2",
					Description: "The URL for the Hexagate API",
				},
			},
			ResourcesMap: map[string]*schema.Resource{
				"hexagate_monitor": resourceMonitor(),
			},
			DataSourcesMap: map[string]*schema.Resource{
				// Add data sources here if needed
			},
		}

		p.ConfigureContextFunc = configure(version)
		return p
	}
}

type ProviderClient struct {
	*HexagateClient
	UserAgent string
}

func configure(version string) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		apiToken := d.Get("api_token").(string)
		apiURL := d.Get("api_url").(string)

		var diags diag.Diagnostics

		// Create a custom User-Agent for API requests
		userAgent := fmt.Sprintf("smartcontracts/hexagate/terraform/%s", version)

		client := &ProviderClient{
			HexagateClient: &HexagateClient{
				APIToken: apiToken,
				BaseURL:  apiURL,
				Client:   &http.Client{},
			},
			UserAgent: userAgent,
		}

		// Test the API connection
		_, err := client.GetAllMonitors()
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Unable to connect to Hexagate API",
				Detail:   fmt.Sprintf("Failed to connect to API: %v", err),
			})
			return nil, diags
		}

		return client, diags
	}
}
