package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccAccessGroupMemberResourceDefault(t *testing.T) {
	workspaceId := testAccWorkspaceId(t)
	userId := os.Getenv("RAILWAY_TEST_USER_ID")

	if userId == "" {
		t.Skip("RAILWAY_TEST_USER_ID must be set for access group member acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccAccessGroupMemberResourceConfigDefault(workspaceId, userId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_access_group_member.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_access_group_member.test", "workspace_id", workspaceId),
					resource.TestCheckResourceAttr("railway_access_group_member.test", "user_id", userId),
					resource.TestCheckResourceAttrPair("railway_access_group_member.test", "access_group_id", "railway_access_group.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_access_group_member.test",
				ImportState:       true,
				ImportStateIdFunc: accessGroupMemberImportIdFunc,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccAccessGroupMemberResourceConfigDefault(workspaceId string, userId string) string {
	return fmt.Sprintf(`
resource "railway_access_group" "test" {
  workspace_id = "%[1]s"
  name = "tf-acc-access-group-member"
  role = "VIEWER"
}

resource "railway_access_group_member" "test" {
  workspace_id = "%[1]s"
  access_group_id = railway_access_group.test.id
  user_id = "%[2]s"
}
`, workspaceId, userId)
}

func accessGroupMemberImportIdFunc(state *terraform.State) (string, error) {
	rawState, ok := state.RootModule().Resources["railway_access_group_member.test"]

	if !ok {
		return "", fmt.Errorf("Resource Not found")
	}

	return fmt.Sprintf("%s:%s:%s", rawState.Primary.Attributes["workspace_id"], rawState.Primary.Attributes["access_group_id"], rawState.Primary.Attributes["user_id"]), nil
}
