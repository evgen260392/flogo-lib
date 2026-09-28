package calculator

import (
	"fmt"
	"reflect"

	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/data/metadata"
)

const (
	ivValueA = "valueA"
	ovValue  = "value"
)

type Settings struct {
	ValueB interface{} `md:"Value B,required"`
	Op     string      `md:"op,allowed(Sum,Sub,Mul,Div)"`
}

type Input struct {
	ValueA interface{} `md:"valueA,required"`
}

type Output struct {
	Value interface{} `md:"value"`
}

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
func (a *Activity) Eval(context activity.Context) (done bool, err error) {
	s := a.settings

	valueA, err := toFloat64(context.GetInput(ivValueA))
	if err != nil {
		return false, fmt.Errorf("input %q: %w", ivValueA, err)
	}
	valueB, err := toFloat64(s.ValueB)
	if err != nil {
		return false, fmt.Errorf("setting %q: %w", "Value B", err)
	}

	var val float64

	switch s.Op {
	case "Sum":
		val = valueA + valueB
	case "Sub":
		val = valueA - valueB
	case "Mul":
		val = valueA * valueB
	case "Div":
		if valueB == 0 {
			return false, fmt.Errorf("cannot divide by zero")
		}
		val = valueA / valueB
	default:
		return false, fmt.Errorf("unsupported operation %q", s.Op)
	}

	err = context.SetOutput(ovValue, val)
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
