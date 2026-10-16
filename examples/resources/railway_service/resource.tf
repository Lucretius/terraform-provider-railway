resource "railway_service" "example" {
  name       = "api"
  project_id = railway_project.example.id
}

resource "railway_service" "staging" {
  name           = "api-staging"
  project_id     = railway_project.example.id
  environment_id = railway_environment.staging.id

  start_command    = "npm start"
  healthcheck_path = "/health"
}
