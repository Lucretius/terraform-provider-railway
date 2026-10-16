package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestReconcileEdgeRules(t *testing.T) {
	configured := `[{"action":"block","expression":"a"}]`

	// Server adds an id and reorders keys: no diff, configured string is kept.
	if got := reconcileEdgeRules(configured, json.RawMessage(`[{"id":"rul_1","expression":"a","action":"block"}]`)); got != configured {
		t.Errorf("expected configured rules to be kept, got %s", got)
	}

	// Real drift: the server value is returned.
	drifted := `[{"id":"rul_1","expression":"b","action":"block"}]`

	if got := reconcileEdgeRules(configured, json.RawMessage(drifted)); got != drifted {
		t.Errorf("expected server rules on drift, got %s", got)
	}

	// Rule removed outside Terraform.
	if got := reconcileEdgeRules(configured, nil); got != "[]" {
		t.Errorf("expected empty rules, got %s", got)
	}
}

func TestAccServiceEdgeRulesResourceDefault(t *testing.T) {
	serviceId := os.Getenv("RAILWAY_TEST_SERVICE_ID")
	environmentId := os.Getenv("RAILWAY_TEST_ENVIRONMENT_ID")
	rules := os.Getenv("RAILWAY_TEST_EDGE_RULES")

	if serviceId == "" || environmentId == "" || rules == "" {
		t.Skip("RAILWAY_TEST_SERVICE_ID, RAILWAY_TEST_ENVIRONMENT_ID and RAILWAY_TEST_EDGE_RULES (a JSON array) must be set for edge rules acceptance tests")
	}

	config := fmt.Sprintf(`
resource "railway_service_edge_rules" "test" {
  service_id = "%s"
  environment_id = "%s"
  edge_rules = %q
}
`, serviceId, environmentId, rules)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing; the plan after apply must be empty (no perpetual diff).
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_edge_rules.test", "id", serviceId+":"+environmentId),
					resource.TestCheckResourceAttr("railway_service_edge_rules.test", "edge_rules", rules),
				),
			},
			// Re-apply the same config: must be a no-op.
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
