package calculator_test

import (
	"testing"

	"github.com/evgen260392/flogo-lib/activity/calculator"
	"github.com/project-flogo/core/api"
	"github.com/stretchr/testify/assert"
)

func TestActivityExecutionThroughFlogoAPI(t *testing.T) {
	act, err := api.NewActivity(&calculator.Activity{}, &calculator.Settings{
		OperandB: 3,
		Operator: "Add",
	})
	if err != nil {
		t.Fatalf("creating activity through Flogo API: %v", err)
	}

	output, err := api.EvalActivity(act, &calculator.Input{OperandA: 9})
	if err != nil {
		t.Fatalf("evaluating activity through Flogo API: %v", err)
	}

	assert.Equal(t, float64(12), output["result"])
}
