#!/usr/bin/env sh

# Access groups can be imported using `workspace_id:access_group_id`.
terraform import railway_access_group.example 00000000-0000-0000-0000-000000000000:00000000-0000-0000-0000-000000000001
