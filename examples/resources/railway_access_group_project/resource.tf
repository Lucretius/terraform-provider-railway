resource "railway_access_group_project" "example" {
  workspace_id    = railway_access_group.example.workspace_id
  access_group_id = railway_access_group.example.id
  project_id      = "00000000-0000-0000-0000-000000000003"
}
