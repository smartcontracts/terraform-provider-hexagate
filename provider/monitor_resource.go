package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &MonitorResource{}
	_ resource.ResourceWithConfigure   = &MonitorResource{}
	_ resource.ResourceWithImportState = &MonitorResource{}
	_ resource.ResourceWithModifyPlan  = &MonitorResource{}
)

// NewMonitorResource is a helper function to simplify the provider implementation.
func NewMonitorResource() resource.Resource {
	return &MonitorResource{}
}

// MonitorResource is the resource implementation.
type MonitorResource struct {
	client *Client
}

// MonitorResourceModel describes the resource data model.
type MonitorResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	MonitorID    types.Int64  `tfsdk:"monitor_id"`
	Description  types.String `tfsdk:"description"`
	Disabled     types.Bool   `tfsdk:"disabled"`
	Entities     types.List   `tfsdk:"entities"`
	MonitorRules types.List   `tfsdk:"monitor_rules"`
	Params       types.String `tfsdk:"params"`
	CreatedBy    types.String `tfsdk:"created_by"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

// EntityModel describes an entity in the monitor.
type EntityModel struct {
	EntityType types.Int64  `tfsdk:"entity_type"`
	Params     types.String `tfsdk:"params"`
}

// MonitorRuleModel describes a rule in the monitor.
type MonitorRuleModel struct {
	ID                 types.Int64  `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Type               types.String `tfsdk:"type"`
	Threshold          types.Int64  `tfsdk:"threshold"`
	NotificationPeriod types.Int64  `tfsdk:"notification_period"`
	Categories         types.List   `tfsdk:"categories"`
	Channels           types.Set    `tfsdk:"channels"`
}

// ChannelModel describes a channel in a monitor rule.
type ChannelModel struct {
	ID     types.Int64  `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Params types.String `tfsdk:"params"`
}

// Configure adds the provider configured client to the resource.
func (r *MonitorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderClient, got: %T", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Metadata returns the resource type name.
func (r *MonitorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitor"
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// It includes the CustomizeDiff logic for the params attribute.
func (r *MonitorResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Retrieve the plan and state
	var plan MonitorResourceModel
	var state MonitorResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// State might not exist during creation
	if !req.State.Raw.IsNull() {
		diags = req.State.Get(ctx, &state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Check if 'params' attribute requires custom diff logic
	paramsPath := path.Root("params")

	var planParams, stateParams types.String

	diags = req.Plan.GetAttribute(ctx, paramsPath, &planParams)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// State might not exist during creation
	if !req.State.Raw.IsNull() {
		diags = req.State.GetAttribute(ctx, paramsPath, &stateParams)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		// If state is null (creation), set stateParams to null explicitly
		// so the checks below don't fail trying to access it.
		stateParams = types.StringNull()
		tflog.Debug(ctx, "Skipping params comparison on create")
		return // Nothing to compare against during creation
	}

	// Proceed only if both plan and state 'params' are known and not null.
	// If plan params is null/unknown, let TF handle it. If state is null (handled above), skip custom diff.
	if planParams.IsNull() || planParams.IsUnknown() || stateParams.IsNull() {
		tflog.Debug(ctx, "Skipping custom diff for params: Plan or State params are null or unknown.")
		return
	}

	planParamsStr := planParams.ValueString()
	stateParamsStr := stateParams.ValueString()

	// If the strings are identical, no need for deep comparison.
	if planParamsStr == stateParamsStr {
		tflog.Debug(ctx, "Params strings are identical, skipping deep comparison.")
		return
	}

	tflog.Debug(ctx, "Params strings differ, performing deep comparison.", map[string]interface{}{
		"plan_params":  planParamsStr,
		"state_params": stateParamsStr,
	})

	var planData, stateData interface{}

	errPlan := json.Unmarshal([]byte(planParamsStr), &planData)
	errState := json.Unmarshal([]byte(stateParamsStr), &stateData)

	if errPlan != nil || errState != nil {
		// If unmarshalling fails, it suggests the strings might not be valid JSON
		// or the format is unexpected. Log this but let Terraform handle the diff as strings.
		tflog.Warn(ctx, "Failed to unmarshal params JSON for comparison; falling back to string diff", map[string]interface{}{
			"plan_error":  errPlan,
			"state_error": errState,
		})
		return
	}

	// Compare the unmarshalled data
	if compareJSONValues(planData, stateData) {
		tflog.Debug(ctx, "Plan params are a subset of state params; suppressing diff.")
		// If the plan data is logically contained within the state data, suppress the diff for 'params'.
		// Use the value read from the state attribute directly
		resp.Plan.SetAttribute(ctx, paramsPath, stateParams)
	} else {
		tflog.Debug(ctx, "Plan params differ logically from state params; allowing diff.")
		// Otherwise, let the plan proceed as is, allowing Terraform to show the diff.
	}
}

// compareJSONValues recursively compares two unmarshalled JSON values (interface{}).
// It returns true if `planValue` is logically contained within `stateValue`,
// meaning all elements in `planValue` exist and match in `stateValue`,
// but `stateValue` can have additional elements.
func compareJSONValues(planValue, stateValue interface{}) bool {
	// Use reflect.DeepEqual for basic types and nil checks
	if reflect.DeepEqual(planValue, stateValue) {
		return true
	}

	planMap, planIsMap := planValue.(map[string]interface{})
	stateMap, stateIsMap := stateValue.(map[string]interface{})

	planSlice, planIsSlice := planValue.([]interface{})
	stateSlice, stateIsSlice := stateValue.([]interface{})

	// Type mismatch (e.g., map vs slice, map vs scalar)
	if planIsMap != stateIsMap || planIsSlice != stateIsSlice {
		return false
	}

	if planIsMap {
		// Compare maps: ensure all keys in planMap exist in stateMap with matching values
		for key, planSubValue := range planMap {
			stateSubValue, ok := stateMap[key]
			if !ok {
				return false // Key missing in state
			}
			if !compareJSONValues(planSubValue, stateSubValue) {
				return false // Values differ recursively
			}
		}
		return true // All plan keys/values found and matched in state
	}

	if planIsSlice {
		// Compare slices: must have the same length and elements must match recursively in order
		// Note: This implements ordered list comparison. If unordered comparison is needed, logic would be more complex.
		if len(planSlice) != len(stateSlice) {
			return false
		}
		for i := range planSlice {
			if !compareJSONValues(planSlice[i], stateSlice[i]) {
				return false
			}
		}
		return true
	}

	// For scalars (string, number, bool, nil), DeepEqual should have caught matches.
	// If we reach here, it means scalars differ.
	return false
}

// Schema defines the schema for the resource.
func (r *MonitorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hexagate monitor",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the monitor",
			},
			"monitor_id": schema.Int64Attribute{
				Optional:    true,
				Description: "The ID of the monitor type",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "A description of the monitor",
			},
			"disabled": schema.BoolAttribute{
				Required:    true,
				Description: "Whether the monitor is disabled",
			},
			"params": schema.StringAttribute{
				Optional:    true,
				Description: "JSON encoded parameters for the monitor",
				Computed:    true,
			},
			"created_by": schema.StringAttribute{
				Computed:    true,
				Description: "The creator of the monitor",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "The creation timestamp",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "The last update timestamp",
			},
		},
		Blocks: map[string]schema.Block{
			"entities": schema.ListNestedBlock{
				Description: "The entities to monitor",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"entity_type": schema.Int64Attribute{
							Required:    true,
							Description: "The type of the entity",
						},
						"params": schema.StringAttribute{
							Required:    true,
							Description: "JSON encoded parameters for the entity",
						},
					},
				},
			},
			"monitor_rules": schema.ListNestedBlock{
				Description: "The rules for the monitor",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Required: true,
						},
						"type": schema.StringAttribute{
							Required: true,
						},
						"threshold": schema.Int64Attribute{
							Required: true,
						},
						"notification_period": schema.Int64Attribute{
							Optional: true,
						},
						"categories": schema.ListAttribute{
							Required:    true,
							ElementType: types.Int64Type,
						},
					},
					Blocks: map[string]schema.Block{
						"channels": schema.SetNestedBlock{
							Description: "The notification channels for the rule",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.Int64Attribute{
										Optional: true,
										Computed: true,
									},
									"name": schema.StringAttribute{
										Required: true,
									},
									"params": schema.StringAttribute{
										Required:    true,
										Description: "JSON encoded parameters for the channel",
										Sensitive:   true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *MonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MonitorResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	monitor := monitorFromModel(ctx, plan)
	if monitor == nil {
		resp.Diagnostics.AddError(
			"Error Creating Monitor",
			"Failed to convert plan to monitor data.",
		)
		return
	}

	result, err := r.client.HexagateClient.CreateMonitor(monitor)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Monitor",
			fmt.Sprintf("Could not create monitor: %s", err),
		)
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(result.ID))

	// Read the response into the state
	diags = r.read(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *MonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MonitorResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = r.read(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *MonitorResource) read(ctx context.Context, state *MonitorResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		diags.AddError(
			"Error Reading Monitor",
			fmt.Sprintf("Could not parse ID: %s", err),
		)
		return diags
	}

	monitor, err := r.client.HexagateClient.GetMonitor(id)
	if err != nil {
		diags.AddError(
			"Error Reading Monitor",
			fmt.Sprintf("Could not read monitor ID %d: %s", id, err),
		)
		return diags
	}

	// Set the ID explicitly
	state.ID = types.StringValue(strconv.Itoa(monitor.ID))

	// Map response to model
	state.Name = types.StringValue(monitor.Name)
	state.MonitorID = types.Int64Value(int64(monitor.MonitorID))
	state.Description = types.StringValue(monitor.Description)
	state.Disabled = types.BoolValue(monitor.Disabled)
	state.CreatedBy = types.StringValue(monitor.CreatedBy)
	state.CreatedAt = types.StringValue(monitor.CreatedAt)
	state.UpdatedAt = types.StringValue(monitor.UpdatedAt)

	// Handle entities
	if monitor.Entities != nil {
		entities := make([]EntityModel, len(monitor.Entities))
		for i, e := range monitor.Entities {
			entityMap := e.(map[string]interface{})
			params, _ := json.Marshal(entityMap["params"])
			entities[i] = EntityModel{
				EntityType: types.Int64Value(int64(entityMap["entity_type"].(float64))),
				Params:     types.StringValue(string(params)),
			}
		}
		state.Entities, diags = types.ListValueFrom(ctx, types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"entity_type": types.Int64Type,
				"params":      types.StringType,
			},
		}, entities)
		if diags.HasError() {
			return diags
		}
	}

	// Handle monitor rules
	if monitor.MonitorRules != nil {
		rules := make([]MonitorRuleModel, len(monitor.MonitorRules))
		for i, r := range monitor.MonitorRules {
			ruleMap := r.(map[string]interface{})

			// Ensure we set the rule ID from the API response
			ruleID := int64(ruleMap["id"].(float64))

			// Handle channels
			channels := make([]ChannelModel, 0)
			if channelsRaw, ok := ruleMap["channels"].([]interface{}); ok {
				for _, ch := range channelsRaw {
					channel := ch.(map[string]interface{})
					params, _ := json.Marshal(channel["params"])
					channels = append(channels, ChannelModel{
						ID:     types.Int64Value(int64(channel["id"].(float64))),
						Name:   types.StringValue(channel["name"].(string)),
						Params: types.StringValue(string(params)),
					})
				}
			}

			// Convert categories
			categories := make([]int64, 0)
			if cats, ok := ruleMap["categories"].([]interface{}); ok {
				for _, c := range cats {
					categories = append(categories, int64(c.(float64)))
				}
			}

			// Convert categories to []attr.Value
			categoryValues := make([]attr.Value, len(categories))
			for i, cat := range categories {
				categoryValues[i] = types.Int64Value(cat)
			}

			channelsValue, diags := types.SetValueFrom(ctx, types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"id":     types.Int64Type,
					"name":   types.StringType,
					"params": types.StringType,
				},
			}, channels)
			if diags.HasError() {
				return diags
			}

			rules[i] = MonitorRuleModel{
				ID:        types.Int64Value(ruleID),
				Name:      types.StringValue(ruleMap["name"].(string)),
				Type:      types.StringValue("notification"),
				Threshold: types.Int64Value(int64(ruleMap["threshold"].(float64))),
			}

			// Set notification_period if it exists in the response
			if notificationPeriod, ok := ruleMap["notification_period"].(float64); ok {
				rules[i].NotificationPeriod = types.Int64Value(int64(notificationPeriod))
			}

			rules[i].Categories = types.ListValueMust(types.Int64Type, categoryValues)
			rules[i].Channels = channelsValue
		}
		state.MonitorRules, diags = types.ListValueFrom(ctx, types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":                  types.Int64Type,
				"name":                types.StringType,
				"type":                types.StringType,
				"threshold":           types.Int64Type,
				"notification_period": types.Int64Type,
				"categories":          types.ListType{ElemType: types.Int64Type},
				"channels": types.SetType{
					ElemType: types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"id":     types.Int64Type,
							"name":   types.StringType,
							"params": types.StringType,
						},
					},
				},
			},
		}, rules)
		if diags.HasError() {
			return diags
		}
	}

	if monitor.Params != nil {
		// Normalize JSON before storing to potentially reduce superficial diffs
		paramsBytes, err := json.Marshal(monitor.Params)
		if err != nil {
			diags.AddError("Error Marshalling Params", fmt.Sprintf("Could not marshal params from API: %s", err))
			return diags
		}
		// Unmarshal and remarshal to get a canonical form (e.g., sorted keys)
		var tempParams interface{}
		if err := json.Unmarshal(paramsBytes, &tempParams); err != nil {
			diags.AddError("Error Unmarshalling Params", fmt.Sprintf("Could not unmarshal params for normalization: %s", err))
			return diags
		}
		normalizedParamsBytes, err := json.Marshal(tempParams)
		if err != nil {
			diags.AddError("Error Re-marshalling Params", fmt.Sprintf("Could not marshal normalized params: %s", err))
			return diags
		}
		state.Params = types.StringValue(string(normalizedParamsBytes))
	} else {
		// Ensure Params is explicitly null if not returned by API
		state.Params = types.StringNull()
	}

	return diags
}

func (r *MonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state MonitorResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var plan MonitorResourceModel
	diags = req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve IDs from state while applying updates from plan
	plan.ID = state.ID

	// Preserve rule IDs if they exist in state
	if !plan.MonitorRules.IsNull() && !state.MonitorRules.IsNull() {
		var planRules, stateRules []MonitorRuleModel
		plan.MonitorRules.ElementsAs(ctx, &planRules, false)
		state.MonitorRules.ElementsAs(ctx, &stateRules, false)

		// Match rules by name and preserve IDs
		for i := range planRules {
			for _, stateRule := range stateRules {
				if planRules[i].Name.ValueString() == stateRule.Name.ValueString() {
					planRules[i].ID = stateRule.ID
					break
				}
			}
		}

		// Create a proper object type for monitor rules
		monitorRuleObject := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":                  types.Int64Type,
				"name":                types.StringType,
				"type":                types.StringType,
				"threshold":           types.Int64Type,
				"notification_period": types.Int64Type,
				"categories":          types.ListType{ElemType: types.Int64Type},
				"channels": types.SetType{
					ElemType: types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"id":     types.Int64Type,
							"name":   types.StringType,
							"params": types.StringType,
						},
					},
				},
			},
		}

		// Update plan.MonitorRules with preserved IDs
		newRules, diags := types.ListValueFrom(ctx, monitorRuleObject, planRules)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.MonitorRules = newRules
	}

	monitor := monitorFromModel(ctx, plan)
	if monitor == nil {
		resp.Diagnostics.AddError(
			"Error Updating Monitor",
			"Failed to convert plan to monitor data.",
		)
		return
	}

	id, err := strconv.Atoi(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Monitor",
			fmt.Sprintf("Could not parse ID: %s", err),
		)
		return
	}

	if err := r.client.HexagateClient.UpdateMonitor(id, monitor); err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Monitor",
			fmt.Sprintf("Could not update monitor ID %d: %s", id, err),
		)
		return
	}

	// Read the response into the state
	diags = r.read(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *MonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MonitorResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Monitor",
			fmt.Sprintf("Could not parse ID: %s", err),
		)
		return
	}

	if err := r.client.HexagateClient.DeleteMonitor(id); err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Monitor",
			fmt.Sprintf("Could not delete monitor ID %d: %s", id, err),
		)
		return
	}
}

func (r *MonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helper function to convert from the model to the API format
func monitorFromModel(ctx context.Context, model MonitorResourceModel) map[string]interface{} {
	monitor := map[string]interface{}{
		"name":          model.Name.ValueString(),
		"disabled":      model.Disabled.ValueBool(),
		"wallets":       []interface{}{},
		"monitor_tags":  []interface{}{},
		"entities_tags": []interface{}{},
	}

	if !model.ID.IsNull() && model.ID.ValueString() != "" {
		monitor["id"] = model.ID.ValueString()
	}

	if !model.MonitorID.IsNull() {
		monitor["monitor_id"] = model.MonitorID.ValueInt64()
	}

	if !model.Description.IsNull() {
		monitor["description"] = model.Description.ValueString()
	}

	// Handle entities
	if !model.Entities.IsNull() {
		var entities []EntityModel
		model.Entities.ElementsAs(ctx, &entities, false)

		apiEntities := make([]map[string]interface{}, len(entities))
		for i, entity := range entities {
			var params map[string]interface{}
			err := json.Unmarshal([]byte(entity.Params.ValueString()), &params)
			if err != nil {
				log.Printf("[ERROR] Error unmarshalling params: %s", err)
				return nil
			}

			apiEntities[i] = map[string]interface{}{
				"entity_type": entity.EntityType.ValueInt64(),
				"params":      params,
			}
		}
		monitor["entities"] = apiEntities
	} else {
		monitor["entities"] = []interface{}{}
	}

	// Handle monitor rules
	if !model.MonitorRules.IsNull() {
		var rules []MonitorRuleModel
		model.MonitorRules.ElementsAs(ctx, &rules, false)

		apiRules := make([]map[string]interface{}, len(rules))
		for i, rule := range rules {
			var channels []ChannelModel
			rule.Channels.ElementsAs(ctx, &channels, false)

			apiChannels := make([]map[string]interface{}, len(channels))
			for j, channel := range channels {
				var params map[string]interface{}
				err := json.Unmarshal([]byte(channel.Params.ValueString()), &params)
				if err != nil {
					log.Printf("[ERROR] Error unmarshalling params: %s", err)
					return nil
				}

				apiChannels[j] = map[string]interface{}{
					"name":   channel.Name.ValueString(),
					"params": params,
				}

				if !channel.ID.IsNull() {
					apiChannels[j]["id"] = channel.ID.ValueInt64()
				}
			}

			var categories []int64
			rule.Categories.ElementsAs(ctx, &categories, false)

			apiRules[i] = map[string]interface{}{
				"name":       rule.Name.ValueString(),
				"type":       rule.Type.ValueString(),
				"threshold":  rule.Threshold.ValueInt64(),
				"categories": categories,
				"channels":   apiChannels,
			}

			// Add notification_period if not null
			if !rule.NotificationPeriod.IsNull() {
				apiRules[i]["notification_period"] = rule.NotificationPeriod.ValueInt64()
			}

			if !rule.ID.IsNull() && rule.ID.ValueInt64() != 0 {
				apiRules[i]["id"] = rule.ID.ValueInt64()
			}
		}
		monitor["monitor_rules"] = apiRules
	}

	// Handle params
	if !model.Params.IsNull() && !model.Params.IsUnknown() {
		var params map[string]interface{}
		// Normalize the JSON string from the config/plan before sending
		paramsStr := model.Params.ValueString()
		var tempParams interface{}
		if err := json.Unmarshal([]byte(paramsStr), &tempParams); err != nil {
			// This might happen if the string is not valid JSON, though schema validation should catch this.
			log.Printf("[WARN] Error unmarshalling params from model for normalization: %s. Sending raw string value.", err)
			// Attempt to send raw if unmarshalling fails, though the API might reject it.
			// Or return nil / add diagnostic? For now, log warning and proceed.
			// It's better to let the API call fail than to silently corrupt data.
			// Let's add a diagnostic and return nil for safety.
			// --> Returning nil seems safer. Add diagnostic elsewhere if needed.
			// log.Printf("[ERROR] Invalid JSON in params attribute: %s", err)
			// Need diags object here to add error. Modify function signature?
			// For now, just return nil as before.
			return nil
		}
		normalizedParamsBytes, err := json.Marshal(tempParams)
		if err != nil {
			log.Printf("[ERROR] Error marshalling normalized params: %s", err)
			return nil
		}
		// Now unmarshal the *normalized* bytes into the map for the API call
		if err := json.Unmarshal(normalizedParamsBytes, &params); err != nil {
			log.Printf("[ERROR] Error unmarshalling normalized params into map: %s", err)
			return nil
		}
		monitor["params"] = params
	} else {
		// If params is null or unknown in the model, don't include it in the API request map.
		// The API might interpret absence differently than `null` or `{}`.
		// Assuming absence means "no change" or "use default".
	}

	return monitor
}
