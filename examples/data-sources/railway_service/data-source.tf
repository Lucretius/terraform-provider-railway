data "railway_service" "api" {
  project_id     = railway_project.example.id
  name           = "api"
  environment_id = data.railway_environment.production.id
}
