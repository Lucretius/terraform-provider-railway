package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccWorkspaceId returns the workspace used by access group acceptance tests.
func testAccWorkspaceId(t *testing.T) string {
	id := os.Getenv("RAILWAY_TEST_WORKSPACE_ID")

	if id == "" {
		t.Skip("RAILWAY_TEST_WORKSPACE_ID must be set for access group acceptance tests")
	}

	return id
}

func TestAccAccessGroupResourceDefault(t *testing.T) {
	workspaceId := testAccWorkspaceId(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccAccessGroupResourceConfigDefault(workspaceId, "tf-acc-access-group", "VIEWER"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_access_group.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_access_group.test", "workspace_id", workspaceId),
					resource.TestCheckResourceAttr("railway_access_group.test", "name", "tf-acc-access-group"),
					resource.TestCheckResourceAttr("railway_access_group.test", "role", "VIEWER"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_access_group.test",
				ImportState:       true,
				ImportStateIdFunc: accessGroupImportIdFunc,
				ImportStateVerify: true,
			},
			// Update name and role
			{
				Config: testAccAccessGroupResourceConfigDefault(workspaceId, "tf-acc-access-group-renamed", "MEMBER"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_access_group.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_access_group.test", "name", "tf-acc-access-group-renamed"),
					resource.TestCheckResourceAttr("railway_access_group.test", "role", "MEMBER"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccAccessGroupResourceConfigDefault(workspaceId string, name string, role string) string {
	return fmt.Sprintf(`
resource "railway_access_group" "test" {
  workspace_id = "%s"
  name = "%s"
  role = "%s"
}
`, workspaceId, name, role)
}

func accessGroupImportIdFunc(state *terraform.State) (string, error) {
	rawState, ok := state.RootModule().Resources["railway_access_group.test"]

	if !ok {
		return "", fmt.Errorf("Resource Not found")
	}

	return fmt.Sprintf("%s:%s", rawState.Primary.Attributes["workspace_id"], rawState.Primary.Attributes["id"]), nil
}
