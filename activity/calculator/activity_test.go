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
	settings := &Settings{ValueB: 2, Op: "Div"}
	mf := mapper.NewFactory(resolve.GetBasicResolver())
	iCtx := test.NewActivityInitContext(settings, mf)

	act, err := New(iCtx)
	assert.Nil(t, err)
	assert.NotNil(t, act)
	assert.NotNil(t, act.Metadata())
}

func TestEvalArithmeticOperations(t *testing.T) {
	tests := []struct {
		name string
		op   string
		a    interface{}
		b    interface{}
		want float64
	}{
		{name: "sum int", op: "Sum", a: int8(9), b: int64(3), want: 12},
		{name: "subtract float", op: "Sub", a: float32(9.5), b: float64(3.25), want: 6.25},
		{name: "multiply unsigned", op: "Mul", a: uint16(9), b: uint(3), want: 27},
		{name: "divide fractional", op: "Div", a: int(9), b: float32(2), want: 4.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := &Settings{ValueB: tt.b, Op: tt.op}
			mf := mapper.NewFactory(resolve.GetBasicResolver())
			iCtx := test.NewActivityInitContext(settings, mf)
			act, err := New(iCtx)
			if err != nil {
				t.Fatalf("creating activity: %v", err)
			}

			tc := test.NewActivityContext(act.Metadata())
			tc.SetInput(ivValueA, tt.a)
			done, err := act.Eval(tc)
			if err != nil {
				t.Fatalf("evaluating activity: %v", err)
			}

			assert.True(t, done)
			assert.Equal(t, tt.want, tc.GetOutput(ovValue))
		})
	}
}

func TestEvalRejectsNonNumericInput(t *testing.T) {
	settings := &Settings{ValueB: 2, Op: "Sum"}
	mf := mapper.NewFactory(resolve.GetBasicResolver())
	iCtx := test.NewActivityInitContext(settings, mf)
	act, err := New(iCtx)
	if err != nil {
		t.Fatalf("creating activity: %v", err)
	}

	tc := test.NewActivityContext(act.Metadata())
	tc.SetInput(ivValueA, "not a number")
	_, err = act.Eval(tc)
	assert.Error(t, err)
}

func TestEvalRejectsDivisionByZero(t *testing.T) {
	settings := &Settings{ValueB: 0, Op: "Div"}
	mf := mapper.NewFactory(resolve.GetBasicResolver())
	iCtx := test.NewActivityInitContext(settings, mf)
	act, err := New(iCtx)
	if err != nil {
		t.Fatalf("creating activity: %v", err)
	}

	tc := test.NewActivityContext(act.Metadata())
	tc.SetInput(ivValueA, 9)
	_, err = act.Eval(tc)
	assert.Error(t, err)
}
