package provider

import (
	"context"
	"fmt"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ProjectDataSource{}
var _ datasource.DataSourceWithConfigure = &ProjectDataSource{}

func NewProjectDataSource() datasource.DataSource {
	return &ProjectDataSource{}
}

type ProjectDataSource struct {
	client *graphql.Client
}

type ProjectDataSourceEnvironmentModel struct {
	Id   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

type ProjectDataSourceModel struct {
	Id                   types.String                        `tfsdk:"id"`
	Name                 types.String                        `tfsdk:"name"`
	Description          types.String                        `tfsdk:"description"`
	IsPublic             types.Bool                          `tfsdk:"is_public"`
	PrDeploys            types.Bool                          `tfsdk:"pr_deploys"`
	WorkspaceId          types.String                        `tfsdk:"workspace_id"`
	DefaultEnvironmentId types.String                        `tfsdk:"default_environment_id"`
	Environments         []ProjectDataSourceEnvironmentModel `tfsdk:"environments"`
}

func (d *ProjectDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *ProjectDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the project.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the project.",
				Computed:            true,
			},
			"is_public": schema.BoolAttribute{
				MarkdownDescription: "Whether the project is public.",
				Computed:            true,
			},
			"pr_deploys": schema.BoolAttribute{
				MarkdownDescription: "Whether pull request deploys are enabled.",
				Computed:            true,
			},
			"workspace_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the workspace the project belongs to.",
				Computed:            true,
			},
			"default_environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the default (oldest) environment.",
				Computed:            true,
			},
			"environments": schema.ListNestedAttribute{
				MarkdownDescription: "Environments of the project.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Identifier of the environment.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of the environment.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *ProjectDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *ProjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *ProjectDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	project, environment, err := defaultEnvironmentForProject(ctx, *d.client, data.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
		return
	}

	data.Name = types.StringValue(project.Name)
	data.Description = types.StringValue(project.Description)
	data.IsPublic = types.BoolValue(project.IsPublic)
	data.PrDeploys = types.BoolValue(project.PrDeploys)
	data.DefaultEnvironmentId = types.StringValue(environment.Id)
	data.WorkspaceId = types.StringNull()

	if project.Workspace != nil {
		data.WorkspaceId = types.StringValue(project.Workspace.Id)
	}

	data.Environments = make([]ProjectDataSourceEnvironmentModel, 0, len(project.Environments.Edges))

	for _, edge := range project.Environments.Edges {
		data.Environments = append(data.Environments, ProjectDataSourceEnvironmentModel{
			Id:   types.StringValue(edge.Node.Id),
			Name: types.StringValue(edge.Node.Name),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
