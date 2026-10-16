#!/usr/bin/env sh

# Access group projects can be imported using `workspace_id:access_group_id:project_id`.
terraform import railway_access_group_project.example 00000000-0000-0000-0000-000000000000:00000000-0000-0000-0000-000000000001:00000000-0000-0000-0000-000000000003
