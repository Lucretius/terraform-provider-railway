package provider

import (
	"context"
	"fmt"
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

var _ resource.Resource = &AccessGroupResource{}
var _ resource.ResourceWithImportState = &AccessGroupResource{}

func NewAccessGroupResource() resource.Resource {
	return &AccessGroupResource{}
}

type AccessGroupResource struct {
	client *graphql.Client
}

type AccessGroupResourceModel struct {
	Id          types.String `tfsdk:"id"`
	WorkspaceId types.String `tfsdk:"workspace_id"`
	Name        types.String `tfsdk:"name"`
	Role        types.String `tfsdk:"role"`
}

// foundAccessGroup is an access group of a workspace along with its membership ids keyed by user id
// and its project attachment ids keyed by project id.
type foundAccessGroup struct {
	AccessGroup
	MemberIds  map[string]string
	ProjectIds map[string]string
}

// findAccessGroup looks up an access group by id in a workspace, returning nil when it doesn't exist.
func findAccessGroup(ctx context.Context, client graphql.Client, workspaceId string, id string) (*foundAccessGroup, error) {
	var after *string

	for {
		response, err := getAccessGroups(ctx, client, workspaceId, after)

		if err != nil {
			return nil, err
		}

		groups := response.Workspace.AccessGroups

		for _, edge := range groups.Edges {
			if edge.Node.Id != id {
				continue
			}

			found := &foundAccessGroup{
				AccessGroup: edge.Node.AccessGroup,
				MemberIds:   map[string]string{},
				ProjectIds:  map[string]string{},
			}

			for _, member := range edge.Node.Members.Edges {
				found.MemberIds[member.Node.UserId] = member.Node.Id
			}

			for _, project := range edge.Node.Projects.Edges {
				found.ProjectIds[project.Node.ProjectId] = project.Node.Id
			}

			return found, nil
		}

		if !groups.PageInfo.HasNextPage {
			return nil, nil
		}

		after = &groups.PageInfo.EndCursor
	}
}

func (r *AccessGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_group"
}

func (r *AccessGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway workspace access group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the access group.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"workspace_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the workspace the access group belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the access group.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"role": schema.StringAttribute{
				MarkdownDescription: "Role granted to the members of the access group. One of `ADMIN`, `MEMBER` or `VIEWER`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(string(ProjectRoleAdmin), string(ProjectRoleMember), string(ProjectRoleViewer)),
				},
			},
		},
	}
}

func (r *AccessGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AccessGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *AccessGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := AccessGroupCreateInput{
		Name:        data.Name.ValueString(),
		Role:        ProjectRole(data.Role.ValueString()),
		WorkspaceId: data.WorkspaceId.ValueString(),
	}

	response, err := createAccessGroup(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create access group, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created an access group")

	group := response.AccessGroupCreate.AccessGroup

	data.Id = types.StringValue(group.Id)
	data.WorkspaceId = types.StringValue(group.WorkspaceId)
	data.Name = types.StringValue(group.Name)
	data.Role = types.StringValue(string(group.Role))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccessGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *AccessGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	group, err := findAccessGroup(ctx, *r.client, data.WorkspaceId.ValueString(), data.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read access group, got error: %s", err))
		return
	}

	if group == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Id = types.StringValue(group.Id)
	data.WorkspaceId = types.StringValue(group.WorkspaceId)
	data.Name = types.StringValue(group.Name)
	data.Role = types.StringValue(string(group.Role))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccessGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *AccessGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := AccessGroupUpdateInput{
		Name: data.Name.ValueString(),
		Role: ProjectRole(data.Role.ValueString()),
	}

	response, err := updateAccessGroup(ctx, *r.client, data.Id.ValueString(), input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update access group, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated an access group")

	group := response.AccessGroupUpdate.AccessGroup

	data.Id = types.StringValue(group.Id)
	data.WorkspaceId = types.StringValue(group.WorkspaceId)
	data.Name = types.StringValue(group.Name)
	data.Role = types.StringValue(string(group.Role))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccessGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *AccessGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := deleteAccessGroup(ctx, *r.client, data.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete access group, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted an access group")
}

func (r *AccessGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: workspace_id:access_group_id. Got: %q", req.ID),
		)

		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
