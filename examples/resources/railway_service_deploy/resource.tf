resource "railway_service_deploy" "example" {
  service_id     = railway_service.example.id
  environment_id = railway_environment.example.id

  # Deploy again whenever one of these changes.
  triggers = {
    start_command = railway_service.example.start_command
    branch        = railway_service.example.source_repo_branch
  }
}
