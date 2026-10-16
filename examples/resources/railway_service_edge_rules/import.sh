#!/usr/bin/env sh

# Edge rules can be imported using `service_id:environment_id`.
# After import, `edge_rules` is read from Railway and shown as a diff against your configuration.
terraform import railway_service_edge_rules.example 00000000-0000-0000-0000-000000000000:00000000-0000-0000-0000-000000000001
