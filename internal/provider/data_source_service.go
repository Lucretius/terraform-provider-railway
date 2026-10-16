package provider

import (
	"context"
	"fmt"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ServiceDataSource{}
var _ datasource.DataSourceWithConfigure = &ServiceDataSource{}
var _ datasource.DataSourceWithConfigValidators = &ServiceDataSource{}

func NewServiceDataSource() datasource.DataSource {
	return &ServiceDataSource{}
}

type ServiceDataSource struct {
	client *graphql.Client
}

type ServiceDataSourceModel struct {
	Id            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	ProjectId     types.String `tfsdk:"project_id"`
	EnvironmentId types.String `tfsdk:"environment_id"`
	SourceImage   types.String `tfsdk:"source_image"`
	SourceRepo    types.String `tfsdk:"source_repo"`
	CronSchedule  types.String `tfsdk:"cron_schedule"`
	RootDirectory types.String `tfsdk:"root_directory"`
	ConfigPath    types.String `tfsdk:"config_path"`
}

func (d *ServiceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (d *ServiceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway service. Look it up by `id`, or by `project_id` and `name`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service. Conflicts with `name`.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the service. Requires `project_id`.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
					stringvalidator.AlsoRequires(path.MatchRoot("project_id")),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project the service belongs to.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment the instance settings are read from. **Default** is the project's default environment.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"source_image": schema.StringAttribute{
				MarkdownDescription: "Source image of the service.",
				Computed:            true,
			},
			"source_repo": schema.StringAttribute{
				MarkdownDescription: "Source repository of the service.",
				Computed:            true,
			},
			"cron_schedule": schema.StringAttribute{
				MarkdownDescription: "Cron schedule of the service.",
				Computed:            true,
			},
			"root_directory": schema.StringAttribute{
				MarkdownDescription: "Root directory of the service.",
				Computed:            true,
			},
			"config_path": schema.StringAttribute{
				MarkdownDescription: "Path to the Railway config file.",
				Computed:            true,
			},
		},
	}
}

func (d *ServiceDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("name")),
	}
}

func (d *ServiceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *ServiceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *ServiceDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !data.Id.IsNull() {
		response, err := getService(ctx, *d.client, data.Id.ValueString())

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service, got error: %s", err))
			return
		}

		data.Name = types.StringValue(response.Service.Name)
		data.ProjectId = types.StringValue(response.Service.ProjectId)
	} else {
		response, err := getProjectServices(ctx, *d.client, data.ProjectId.ValueString())

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read services, got error: %s", err))
			return
		}

		found := false

		for _, edge := range response.Project.Services.Edges {
			if edge.Node.Name == data.Name.ValueString() {
				data.Id = types.StringValue(edge.Node.Id)
				found = true
				break
			}
		}

		if !found {
			resp.Diagnostics.AddError("Not Found", fmt.Sprintf("No service named %q in project %s", data.Name.ValueString(), data.ProjectId.ValueString()))
			return
		}
	}

	environmentId, err := resolveEnvironmentId(ctx, *d.client, data.ProjectId.ValueString(), data.EnvironmentId)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to resolve environment, got error: %s", err))
		return
	}

	data.EnvironmentId = types.StringValue(environmentId)

	instance, err := getServiceInstance(ctx, *d.client, environmentId, data.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service settings, got error: %s", err))
		return
	}

	data.CronSchedule = types.StringPointerValue(instance.ServiceInstance.CronSchedule)
	data.RootDirectory = types.StringPointerValue(instance.ServiceInstance.RootDirectory)
	data.ConfigPath = types.StringPointerValue(instance.ServiceInstance.RailwayConfigFile)
	data.SourceImage = types.StringNull()
	data.SourceRepo = types.StringNull()

	if source := instance.ServiceInstance.Source; source != nil {
		data.SourceImage = types.StringPointerValue(source.Image)
		data.SourceRepo = types.StringPointerValue(source.Repo)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
