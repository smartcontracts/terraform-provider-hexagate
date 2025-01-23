package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceMonitor() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMonitorCreate,
		ReadContext:   resourceMonitorRead,
		UpdateContext: resourceMonitorUpdate,
		DeleteContext: resourceMonitorDelete,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"monitor_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(1, 57),
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"entities": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"entity_type": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"params": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								if _, err := json.Marshal(v); err != nil {
									errors = append(errors, fmt.Errorf("%q must be valid JSON: %s", k, err))
								}
								return
							},
						},
					},
				},
			},
			"disabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"monitor_rules": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"type": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice([]string{"notification"}, false),
						},
						"threshold": {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validation.IntInSlice([]int{10, 30, 50, 70, 90}),
						},
						"categories": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Schema{
								Type:         schema.TypeInt,
								ValidateFunc: validation.IntBetween(1, 7),
							},
						},
						"channels": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:     schema.TypeInt,
										Optional: true,
										Computed: true,
									},
									"name": {
										Type:     schema.TypeString,
										Required: true,
									},
									"params": {
										Type:     schema.TypeString,
										Required: true,
										ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
											if _, err := json.Marshal(v); err != nil {
												errors = append(errors, fmt.Errorf("%q must be valid JSON: %s", k, err))
											}
											return
										},
									},
								},
							},
						},
					},
				},
			},
			"params": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					if _, err := json.Marshal(v); err != nil {
						errors = append(errors, fmt.Errorf("%q must be valid JSON: %s", k, err))
					}
					return
				},
			},
			// Read-only fields
			"created_by": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"created_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"updated_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceMonitorCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*ProviderClient).HexagateClient

	monitor := monitorFromResourceData(d)
	if monitor == nil {
		return diag.FromErr(fmt.Errorf("failed to build monitor from resource data"))
	}

	resp, err := client.CreateMonitor(monitor)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(resp.ID))
	return resourceMonitorRead(ctx, d, m)
}

func resourceMonitorRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*ProviderClient).HexagateClient

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	monitor, err := client.GetMonitor(id)
	if err != nil {
		return diag.FromErr(err)
	}

	// Map response to resource data
	d.Set("name", monitor.Name)
	d.Set("id", monitor.ID)
	d.Set("monitor_id", monitor.MonitorID)
	d.Set("description", monitor.Description)
	d.Set("entities", monitor.Entities)
	d.Set("disabled", monitor.Disabled)
	d.Set("params", monitor.Params)
	d.Set("created_by", monitor.CreatedBy)
	d.Set("created_at", monitor.CreatedAt)
	d.Set("updated_at", monitor.UpdatedAt)

	if monitor.MonitorRules != nil {
		rules := make([]map[string]interface{}, len(monitor.MonitorRules))
		for i, rule := range monitor.MonitorRules {
			ruleMap := rule.(map[string]interface{})

			// Convert channels to match schema
			channels := make([]map[string]interface{}, 0)
			if channelsRaw, ok := ruleMap["channels"].([]interface{}); ok {
				for _, ch := range channelsRaw {
					channel := ch.(map[string]interface{})

					// Normalize params by removing null fields
					paramsMap := make(map[string]interface{})
					if params, ok := channel["params"].(map[string]interface{}); ok {
						for k, v := range params {
							if v != nil {
								paramsMap[k] = v
							}
						}
					}

					// Ensure params is a JSON string as per schema
					paramsJSON, _ := json.Marshal(paramsMap)

					channels = append(channels, map[string]interface{}{
						"id":     channel["id"].(float64),
						"name":   channel["name"].(string),
						"params": string(paramsJSON),
					})
				}
			}

			rules[i] = map[string]interface{}{
				"id":         ruleMap["id"].(float64),
				"name":       ruleMap["name"].(string),
				"type":       "notification", // This appears to be fixed based on your schema
				"threshold":  ruleMap["threshold"].(float64),
				"categories": ruleMap["categories"].([]interface{}),
				"channels":   channels,
			}
		}
		d.Set("monitor_rules", rules)
	}

	return nil
}

func resourceMonitorUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*ProviderClient).HexagateClient

	monitor := monitorFromResourceData(d)
	if monitor == nil {
		return diag.FromErr(fmt.Errorf("failed to build monitor from resource data"))
	}

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	monitor["id"] = id

	if err := client.UpdateMonitor(id, monitor); err != nil {
		return diag.FromErr(err)
	}

	return resourceMonitorRead(ctx, d, m)
}

func resourceMonitorDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*ProviderClient).HexagateClient

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := client.DeleteMonitor(id); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}

func monitorFromResourceData(d *schema.ResourceData) map[string]interface{} {
	monitor := map[string]interface{}{
		"name":          d.Get("name").(string),
		"disabled":      d.Get("disabled").(bool),
		"wallets":       []interface{}{},
		"monitor_tags":  []interface{}{},
		"entities_tags": []interface{}{},
	}

	// Handle optional ID for updates
	if id, ok := d.GetOk("id"); ok {
		monitor["id"], _ = strconv.Atoi(id.(string))
	}

	if v, ok := d.GetOk("monitor_id"); ok {
		monitor["monitor_id"] = v.(int)
	}

	if v, ok := d.GetOk("description"); ok {
		monitor["description"] = v.(string)
	}

	if v, ok := d.GetOk("entities"); ok {
		entitiesList := v.([]interface{})
		entities := make([]map[string]interface{}, len(entitiesList))

		for i, entity := range entitiesList {
			entityMap := entity.(map[string]interface{})
			processedEntity := map[string]interface{}{
				"entity_type": entityMap["entity_type"],
			}

			if paramsStr, ok := entityMap["params"].(string); ok {
				var paramsMap map[string]interface{}
				if err := json.Unmarshal([]byte(paramsStr), &paramsMap); err != nil {
					return nil
				}
				processedEntity["params"] = paramsMap
			}

			entities[i] = processedEntity
		}
		monitor["entities"] = entities
	} else {
		monitor["entities"] = []interface{}{}
	}

	if v, ok := d.GetOk("monitor_rules"); ok {
		rulesList := v.([]interface{})
		rules := make([]map[string]interface{}, len(rulesList))

		for i, rule := range rulesList {
			ruleMap := rule.(map[string]interface{})

			categoriesRaw := ruleMap["categories"].([]interface{})
			categories := make([]int, len(categoriesRaw))
			for i, cat := range categoriesRaw {
				categories[i] = cat.(int)
			}

			monitorRule := map[string]interface{}{
				"name":       ruleMap["name"].(string),
				"type":       ruleMap["type"].(string),
				"threshold":  ruleMap["threshold"].(int),
				"categories": categories,
			}

			if id, ok := ruleMap["id"]; ok {
				if id.(int) != 0 {
					monitorRule["id"] = id.(int)
				}
			}

			if channels, ok := ruleMap["channels"].([]interface{}); ok {
				monitorRule["channels"] = make([]map[string]interface{}, len(channels))

				for j, ch := range channels {
					channelMap := ch.(map[string]interface{})
					channel := map[string]interface{}{
						"name": channelMap["name"].(string),
					}

					if id, ok := channelMap["id"]; ok {
						channel["id"] = id.(int)
					}

					if paramsStr, ok := channelMap["params"].(string); ok {
						var params map[string]interface{}
						if err := json.Unmarshal([]byte(paramsStr), &params); err != nil {
							return nil // This will be caught by the calling function
						}
						channel["params"] = params
					}

					monitorRule["channels"].([]map[string]interface{})[j] = channel
				}
			}

			rules[i] = monitorRule
		}
		monitor["monitor_rules"] = rules
	}

	if v, ok := d.GetOk("params"); ok {
		var paramsMap map[string]interface{}
		if err := json.Unmarshal([]byte(v.(string)), &paramsMap); err != nil {
			return nil // This will be caught by the calling function
		}
		monitor["params"] = paramsMap
	}

	return monitor
}
