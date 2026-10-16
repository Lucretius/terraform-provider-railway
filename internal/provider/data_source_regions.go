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

var _ datasource.DataSource = &RegionsDataSource{}
var _ datasource.DataSourceWithConfigure = &RegionsDataSource{}

func NewRegionsDataSource() datasource.DataSource {
	return &RegionsDataSource{}
}

type RegionsDataSource struct {
	client *graphql.Client
}

type RegionsDataSourceRegionModel struct {
	Id       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Country  types.String `tfsdk:"country"`
	Location types.String `tfsdk:"location"`
}

type RegionsDataSourceModel struct {
	ProjectId types.String                   `tfsdk:"project_id"`
	Regions   []RegionsDataSourceRegionModel `tfsdk:"regions"`
}

func (d *RegionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_regions"
}

func (d *RegionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Regions services can be deployed in.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project, to also include regions specific to its workspace.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"regions": schema.ListNestedAttribute{
				MarkdownDescription: "Available regions.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Identifier of the region, to be used in `regions` of `railway_service`.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of the region.",
							Computed:            true,
						},
						"country": schema.StringAttribute{
							MarkdownDescription: "Country of the region.",
							Computed:            true,
						},
						"location": schema.StringAttribute{
							MarkdownDescription: "Location of the region.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *RegionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *RegionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *RegionsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	response, err := getRegions(ctx, *d.client, data.ProjectId.ValueStringPointer())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read regions, got error: %s", err))
		return
	}

	data.Regions = make([]RegionsDataSourceRegionModel, 0, len(response.Regions))

	for _, region := range response.Regions {
		data.Regions = append(data.Regions, RegionsDataSourceRegionModel{
			Id:       types.StringPointerValue(region.Id),
			Name:     types.StringValue(region.Name),
			Country:  types.StringValue(region.Country),
			Location: types.StringValue(region.Location),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
