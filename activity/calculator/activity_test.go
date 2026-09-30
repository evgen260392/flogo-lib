package calculator

import (
	"testing"

	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/data/mapper"
	"github.com/project-flogo/core/data/resolve"
	"github.com/project-flogo/core/support/test"
	"github.com/stretchr/testify/assert"
)

func TestRegister(t *testing.T) {

	ref := activity.GetRef(&Activity{})
	act := activity.Get(ref)

	assert.NotNil(t, act)
}

func TestNew(t *testing.T) {
	settings := &Settings{OperandB: 2, Operator: "Div"}
	mf := mapper.NewFactory(resolve.GetBasicResolver())
	iCtx := test.NewActivityInitContext(settings, mf)

	act, err := New(iCtx)
	assert.Nil(t, err)
	assert.NotNil(t, act)
	assert.NotNil(t, act.Metadata())
}

func TestEvalArithmeticOperations(t *testing.T) {
	tests := []struct {
		name     string
		operator string
		operandA interface{}
		operandB interface{}
		want     float64
	}{
		{name: "add int", operator: "Add", operandA: int8(9), operandB: int64(3), want: 12},
		{name: "subtract float", operator: "Sub", operandA: float32(9.5), operandB: float64(3.25), want: 6.25},
		{name: "multiply unsigned", operator: "Mul", operandA: uint16(9), operandB: uint(3), want: 27},
		{name: "divide fractional", operator: "Div", operandA: int(9), operandB: float32(2), want: 4.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := &Settings{OperandB: tt.operandB, Operator: tt.operator}
			mf := mapper.NewFactory(resolve.GetBasicResolver())
			iCtx := test.NewActivityInitContext(settings, mf)
			act, err := New(iCtx)
			if err != nil {
				t.Fatalf("creating activity: %v", err)
			}

			tc := test.NewActivityContext(act.Metadata())
			tc.SetInput(ivOperandA, tt.operandA)
			done, err := act.Eval(tc)
			if err != nil {
				t.Fatalf("evaluating activity: %v", err)
			}

			assert.True(t, done)
			assert.Equal(t, tt.want, tc.GetOutput(ovResult))
		})
	}
}

func TestEvalRejectsNonNumericInput(t *testing.T) {
	settings := &Settings{OperandB: 2, Operator: "Add"}
	mf := mapper.NewFactory(resolve.GetBasicResolver())
	iCtx := test.NewActivityInitContext(settings, mf)
	act, err := New(iCtx)
	if err != nil {
		t.Fatalf("creating activity: %v", err)
	}

	tc := test.NewActivityContext(act.Metadata())
	tc.SetInput(ivOperandA, "not a number")
	_, err = act.Eval(tc)
	assert.Error(t, err)
}

func TestEvalRejectsDivisionByZero(t *testing.T) {
	settings := &Settings{OperandB: 0, Operator: "Div"}
	mf := mapper.NewFactory(resolve.GetBasicResolver())
	iCtx := test.NewActivityInitContext(settings, mf)
	act, err := New(iCtx)
	if err != nil {
		t.Fatalf("creating activity: %v", err)
	}

	tc := test.NewActivityContext(act.Metadata())
	tc.SetInput(ivOperandA, 9)
	_, err = act.Eval(tc)
	assert.Error(t, err)
}
