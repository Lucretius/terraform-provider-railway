package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccDataSourceConfig = `
resource "railway_environment" "test" {
  name       = "data-source"
  project_id = "0bb01547-570d-4109-a5e8-138691f6a2d1"
}

resource "railway_service" "test" {
  name       = "data-source-app"
  project_id = "0bb01547-570d-4109-a5e8-138691f6a2d1"
}

data "railway_environment" "by_name" {
  project_id = railway_environment.test.project_id
  name       = railway_environment.test.name
}

data "railway_environment" "by_id" {
  id = railway_environment.test.id
}

data "railway_service" "by_name" {
  project_id = railway_service.test.project_id
  name       = railway_service.test.name
}

data "railway_service" "by_id" {
  id             = railway_service.test.id
  environment_id = railway_environment.test.id
}

data "railway_project" "test" {
  id = railway_service.test.project_id
}

data "railway_regions" "test" {
  project_id = railway_service.test.project_id
}
`

func TestAccDataSources(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.railway_environment.by_name", "id", "railway_environment.test", "id"),
					resource.TestCheckResourceAttrPair("data.railway_environment.by_id", "name", "railway_environment.test", "name"),
					resource.TestCheckResourceAttrPair("data.railway_environment.by_id", "project_id", "railway_environment.test", "project_id"),
					resource.TestCheckResourceAttrPair("data.railway_service.by_name", "id", "railway_service.test", "id"),
					resource.TestCheckResourceAttrPair("data.railway_service.by_id", "name", "railway_service.test", "name"),
					resource.TestCheckResourceAttrPair("data.railway_service.by_id", "environment_id", "railway_environment.test", "id"),
					resource.TestCheckResourceAttr("data.railway_project.test", "id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestMatchResourceAttr("data.railway_project.test", "default_environment_id", uuidRegex()),
					resource.TestMatchResourceAttr("data.railway_regions.test", "regions.#", regexp.MustCompile(`^[1-9]\d*$`)),
				),
			},
		},
	})
}

func TestAccDataSourceEnvironmentNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "railway_environment" "test" {
  project_id = "0bb01547-570d-4109-a5e8-138691f6a2d1"
  name       = "does-not-exist"
}
`,
				ExpectError: regexp.MustCompile("No environment named"),
			},
			{
				Config: `
data "railway_environment" "test" {
  name = "production"
}
`,
				ExpectError: regexp.MustCompile("project_id"),
			},
		},
	})
}
