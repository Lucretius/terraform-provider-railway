package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildServiceInstanceInput(t *testing.T) {
	ctx := context.Background()

	data := &ServiceResourceModel{
		StartCommand:            types.StringValue("npm start"),
		Builder:                 types.StringValue("RAILPACK"),
		RestartPolicyType:       types.StringValue("ON_FAILURE"),
		RestartPolicyMaxRetries: types.Int64Value(3),
		WatchPatterns:           types.ListValueMust(types.StringType, []attr.Value{types.StringValue("src/**")}),
		// everything else unset, as the framework would hand it over
		BuildCommand:       types.StringNull(),
		DockerfilePath:     types.StringNull(),
		HealthcheckPath:    types.StringNull(),
		HealthcheckTimeout: types.Int64Null(),
		PreDeployCommand:   types.ListNull(types.StringType),
		SleepApplication:   types.BoolNull(),
	}

	input, diags := buildServiceInstanceInput(ctx, data, nil)

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if input.StartCommand == nil || *input.StartCommand != "npm start" {
		t.Errorf("start command not set: %v", input.StartCommand)
	}

	if input.Builder == nil || *input.Builder != BuilderRailpack {
		t.Errorf("builder not set: %v", input.Builder)
	}

	if input.RestartPolicyMaxRetries == nil || *input.RestartPolicyMaxRetries != 3 {
		t.Errorf("max retries not set: %v", input.RestartPolicyMaxRetries)
	}

	if input.WatchPatterns == nil || len(*input.WatchPatterns) != 1 {
		t.Errorf("watch patterns not set: %v", input.WatchPatterns)
	}

	if input.BuildCommand != nil || input.HealthcheckTimeout != nil || input.PreDeployCommand != nil || input.SleepApplication != nil {
		t.Errorf("unset fields must stay nil: %+v", input)
	}
}
