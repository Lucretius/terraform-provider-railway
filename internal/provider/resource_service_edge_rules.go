package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &ServiceEdgeRulesResource{}
var _ resource.ResourceWithImportState = &ServiceEdgeRulesResource{}

func NewServiceEdgeRulesResource() resource.Resource {
	return &ServiceEdgeRulesResource{}
}

type ServiceEdgeRulesResource struct {
	client *graphql.Client
}

type ServiceEdgeRulesResourceModel struct {
	Id            types.String `tfsdk:"id"`
	ServiceId     types.String `tfsdk:"service_id"`
	EnvironmentId types.String `tfsdk:"environment_id"`
	EdgeRules     types.String `tfsdk:"edge_rules"`
}

// jsonArrayValidator checks that a string is a JSON array.
type jsonArrayValidator struct{}

func (jsonArrayValidator) Description(ctx context.Context) string {
	return "value must be a JSON array"
}

func (v jsonArrayValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (jsonArrayValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	var rules []interface{}

	if err := json.Unmarshal([]byte(req.ConfigValue.ValueString()), &rules); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid edge rules", fmt.Sprintf("Expected a JSON array of rules, got error: %s", err))
	}
}

// jsonCovers reports whether every field of want is present in have with an equal value. Extra
// fields in have are ignored, so fields the server adds (rule ids, defaults) don't count as a difference.
func jsonCovers(want, have interface{}) bool {
	switch w := want.(type) {
	case map[string]interface{}:
		h, ok := have.(map[string]interface{})

		if !ok {
			return false
		}

		for key, value := range w {
			if hv, ok := h[key]; !ok || !jsonCovers(value, hv) {
				return false
			}
		}

		return true
	case []interface{}:
		h, ok := have.([]interface{})

		if !ok || len(h) != len(w) {
			return false
		}

		for i := range w {
			if !jsonCovers(w[i], h[i]) {
				return false
			}
		}

		return true
	default:
		return reflect.DeepEqual(want, have)
	}
}

// reconcileEdgeRules returns the configured rules when the server's rules are equivalent to
// them (avoiding a perpetual diff), and the server's rules otherwise so real drift is shown.
func reconcileEdgeRules(configured string, server json.RawMessage) string {
	serverRules := string(server)

	if serverRules == "" || serverRules == "null" {
		serverRules = "[]"
	}

	var want, have interface{}

	if json.Unmarshal([]byte(configured), &want) == nil && json.Unmarshal([]byte(serverRules), &have) == nil && jsonCovers(want, have) {
		return configured
	}

	return serverRules
}

func (r *ServiceEdgeRulesResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_edge_rules"
}

func (r *ServiceEdgeRulesResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway service edge rules. Replaces the whole set of edge rules of a service in an environment; destroying the resource clears them.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the edge rules, in the format `service_id:environment_id`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service the edge rules belong to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment the edge rules apply to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"edge_rules": schema.StringAttribute{
				MarkdownDescription: "JSON array of edge rules, usually built with `jsonencode`. Rules without an `id` get one assigned by Railway; fields Railway adds are not treated as changes.",
				Required:            true,
				Validators: []validator.String{
					jsonArrayValidator{},
				},
			},
		},
	}
}

func (r *ServiceEdgeRulesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*graphql.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *graphql.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *ServiceEdgeRulesResource) apply(ctx context.Context, data *ServiceEdgeRulesResourceModel) error {
	_, err := updateServiceEdgeRules(ctx, *r.client, data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), json.RawMessage(data.EdgeRules.ValueString()))

	if err != nil {
		return err
	}

	data.Id = types.StringValue(fmt.Sprintf("%s:%s", data.ServiceId.ValueString(), data.EnvironmentId.ValueString()))

	return nil
}

func (r *ServiceEdgeRulesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ServiceEdgeRulesResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(ctx, data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set edge rules, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "set service edge rules")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceEdgeRulesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ServiceEdgeRulesResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	response, err := getServiceEdgeRules(ctx, *r.client, data.EnvironmentId.ValueString(), data.ServiceId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read edge rules, got error: %s", err))
		return
	}

	data.Id = types.StringValue(fmt.Sprintf("%s:%s", data.ServiceId.ValueString(), data.EnvironmentId.ValueString()))
	data.EdgeRules = types.StringValue(reconcileEdgeRules(data.EdgeRules.ValueString(), response.ServiceInstance.EdgeConfig.EdgeRules))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceEdgeRulesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *ServiceEdgeRulesResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(ctx, data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update edge rules, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated service edge rules")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceEdgeRulesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ServiceEdgeRulesResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// A nil RawMessage is sent as null, which clears the rules.
	_, err := updateServiceEdgeRules(ctx, *r.client, data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), nil)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to clear edge rules, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "cleared service edge rules")
}

func (r *ServiceEdgeRulesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: service_id:environment_id. Got: %q", req.ID),
		)

		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
}
