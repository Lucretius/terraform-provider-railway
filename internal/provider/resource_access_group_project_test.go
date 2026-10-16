package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccAccessGroupProjectResourceDefault(t *testing.T) {
	workspaceId := testAccWorkspaceId(t)
	projectId := os.Getenv("RAILWAY_TEST_PROJECT_ID")

	if projectId == "" {
		t.Skip("RAILWAY_TEST_PROJECT_ID must be set for access group project acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccAccessGroupProjectResourceConfigDefault(workspaceId, projectId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_access_group_project.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_access_group_project.test", "workspace_id", workspaceId),
					resource.TestCheckResourceAttr("railway_access_group_project.test", "project_id", projectId),
					resource.TestCheckResourceAttrPair("railway_access_group_project.test", "access_group_id", "railway_access_group.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_access_group_project.test",
				ImportState:       true,
				ImportStateIdFunc: accessGroupProjectImportIdFunc,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccAccessGroupProjectResourceConfigDefault(workspaceId string, projectId string) string {
	return fmt.Sprintf(`
resource "railway_access_group" "test" {
  workspace_id = "%[1]s"
  name = "tf-acc-access-group-project"
  role = "VIEWER"
}

resource "railway_access_group_project" "test" {
  workspace_id = "%[1]s"
  access_group_id = railway_access_group.test.id
  project_id = "%[2]s"
}
`, workspaceId, projectId)
}

func accessGroupProjectImportIdFunc(state *terraform.State) (string, error) {
	rawState, ok := state.RootModule().Resources["railway_access_group_project.test"]

	if !ok {
		return "", fmt.Errorf("Resource Not found")
	}

	return fmt.Sprintf("%s:%s:%s", rawState.Primary.Attributes["workspace_id"], rawState.Primary.Attributes["access_group_id"], rawState.Primary.Attributes["project_id"]), nil
}
