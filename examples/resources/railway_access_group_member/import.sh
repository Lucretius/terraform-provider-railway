#!/usr/bin/env sh

# Access group members can be imported using `workspace_id:access_group_id:user_id`.
terraform import railway_access_group_member.example 00000000-0000-0000-0000-000000000000:00000000-0000-0000-0000-000000000001:00000000-0000-0000-0000-000000000002
