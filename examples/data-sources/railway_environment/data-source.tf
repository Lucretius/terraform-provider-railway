data "railway_environment" "production" {
  project_id = railway_project.example.id
  name       = "production"
}
