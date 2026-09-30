package calculator

import (
	"fmt"
	"reflect"

	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/data/metadata"
)

func init() {
	_ = activity.Register(&Activity{}, New)
}

var activityMd = activity.ToMetadata(&Settings{}, &Input{}, &Output{})

type Activity struct {
	settings *Settings
}

func New(ctx activity.InitContext) (activity.Activity, error) {
	settings := &Settings{}
	err := metadata.MapToStruct(ctx.Settings(), settings, true)
	if err != nil {
		return nil, err
	}

	return &Activity{settings: settings}, nil
}

// Metadata implements activity.Activity.Metadata
func (a *Activity) Metadata() *activity.Metadata {
	return activityMd
}

// Eval implements activity.Activity.Eval
func (a *Activity) Eval(ctx activity.Context) (done bool, err error) {
	s := a.settings
	input := &Input{}

	err = ctx.GetInputObject(input)
	if err != nil {
		return false, fmt.Errorf("input %q: %w", ivOperandA, err)
	}
	operandA, err := toFloat64(input.OperandA)
	if err != nil {
		return false, fmt.Errorf("input %q: %w", ivOperandA, err)
	}
	operandB, err := toFloat64(s.OperandB)
	if err != nil {
		return false, fmt.Errorf("setting %q: %w", "Operand B", err)
	}

	var result float64

	switch s.Operator {
	case "Add":
		result = operandA + operandB
	case "Sub":
		result = operandA - operandB
	case "Mul":
		result = operandA * operandB
	case "Div":
		if operandB == 0 {
			return false, fmt.Errorf("cannot divide by zero")
		}
		result = operandA / operandB
	default:
		return false, fmt.Errorf("unsupported operation %q", s.Operator)
	}

	err = ctx.SetOutputObject(&Output{Result: result})
	if err != nil {
		return false, err
	}

	return true, nil
}

func toFloat64(value interface{}) (float64, error) {
	if value == nil {
		return 0, fmt.Errorf("value is nil, expected a number")
	}

	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(reflected.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(reflected.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return reflected.Convert(reflect.TypeOf(float64(0))).Float(), nil
	default:
		return 0, fmt.Errorf("value %v has non-numeric type %T", value, value)
	}
}
