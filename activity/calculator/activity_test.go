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
	settings := &Settings{ValueA: 8, ValueB: 2, Op: "Div"}
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
		a    int32
		b    int32
		want int32
	}{
		{name: "sum", op: "Sum", a: 9, b: 3, want: 12},
		{name: "subtract", op: "Sub", a: 9, b: 3, want: 6},
		{name: "multiply", op: "Mul", a: 9, b: 3, want: 27},
		{name: "divide", op: "Div", a: 9, b: 3, want: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := &Settings{ValueA: tt.a, ValueB: tt.b, Op: tt.op}
			mf := mapper.NewFactory(resolve.GetBasicResolver())
			iCtx := test.NewActivityInitContext(settings, mf)
			act, err := New(iCtx)
			if err != nil {
				t.Fatalf("creating activity: %v", err)
			}

			tc := test.NewActivityContext(act.Metadata())
			done, err := act.Eval(tc)
			if err != nil {
				t.Fatalf("evaluating activity: %v", err)
			}

			assert.True(t, done)
			assert.Equal(t, tt.want, tc.GetOutput(ovValue))
		})
	}
}
