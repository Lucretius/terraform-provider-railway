# The rule format is defined by Railway; check it with the `validateServiceEdgeRules` API
# or the dashboard. The rule below is illustrative.
resource "railway_service_edge_rules" "example" {
  service_id     = "00000000-0000-0000-0000-000000000000"
  environment_id = "00000000-0000-0000-0000-000000000001"
  edge_rules = jsonencode([
    {
      expression = "http.request.uri.path eq \"/admin\""
      action     = "block"
    }
  ])
}
