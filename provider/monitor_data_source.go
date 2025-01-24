package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &MonitorDataSource{}

func NewMonitorDataSource() datasource.DataSource {
	return &MonitorDataSource{}
}

type MonitorDataSource struct {
	client *Client
}

func (d *MonitorDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *ProviderClient, got: %T", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *MonitorDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitor"
}

func (d *MonitorDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a Hexagate monitor by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Monitor identifier",
			},
			// Reuse the same attributes as the resource, but make them computed
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the monitor.",
			},
			"monitor_id": schema.Int64Attribute{
				Computed:    true,
				Description: "The ID of the monitor type.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "A description of the monitor.",
			},
			"disabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the monitor is disabled.",
			},
			"entities": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The entities to monitor.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"entity_type": schema.Int64Attribute{
							Computed:    true,
							Description: "The type of the entity.",
						},
						"params": schema.StringAttribute{
							Computed:    true,
							Description: "JSON encoded parameters for the entity.",
						},
					},
				},
			},
			"monitor_rules": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The rules for the monitor.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the rule.",
						},
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "The type of the rule.",
						},
						"threshold": schema.Int64Attribute{
							Computed:    true,
							Description: "The threshold for the rule.",
						},
						"categories": schema.ListAttribute{
							Computed:    true,
							Description: "The categories for the rule.",
							ElementType: types.Int64Type,
						},
						"channels": schema.ListNestedAttribute{
							Computed:    true,
							Description: "The notification channels for the rule.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.Int64Attribute{
										Computed: true,
									},
									"name": schema.StringAttribute{
										Computed:    true,
										Description: "The name of the channel.",
									},
									"params": schema.StringAttribute{
										Computed:    true,
										Description: "JSON encoded parameters for the channel.",
									},
								},
							},
						},
					},
				},
			},
			"params": schema.StringAttribute{
				Computed:    true,
				Description: "JSON encoded parameters for the monitor.",
			},
			"created_by": schema.StringAttribute{
				Computed:    true,
				Description: "The creator of the monitor.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "The creation timestamp.",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "The last update timestamp.",
			},
		},
	}
}

func (d *MonitorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state MonitorResourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Reuse the read function from the resource
	resource := MonitorResource{client: d.client}
	diags = resource.read(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
