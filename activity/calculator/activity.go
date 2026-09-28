package calculator

import (
	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/data/metadata"
)

const (
	ovValue = "value"
)


type Settings struct {
	ValueA 	int32 `md:"Value A,required"`
	ValueB 	int32 `md:"Value B,required"`
	Op 		string `md:"op,allowed(Sum,Sub,Mul,Div)"`  
}

type Output struct {
	Value int `md:"value"`
}

// 
func init() {
	_ = activity.Register(&Activity{}, New)
}

var activityMd = activity.ToMetadata(&Settings{}, &Output{})

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

	var val int32

	switch s.Op {
	case "Sum":
		val = s.ValueA + s.ValueB 
	case "Sub":
		val = s.ValueA - s.ValueB 
	case "Mul":
		val = s.ValueA * s.ValueB 
	case "Div":
		val = s.ValueA / s.ValueB 
	}

	err = context.SetOutput(ovValue, int32(val))
	if err != nil {
		return false, err
	}

	return true, nil
}