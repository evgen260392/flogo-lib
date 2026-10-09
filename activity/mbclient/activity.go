package mbclient

import (
	"fmt"

	"github.com/project-flogo/core/activity"
)

func init() {
	_ = activity.Register(&Activity{}, New)
}

var activityMd = activity.ToMetadata(&Input{}, &Output{})

type Activity struct{}

func New(activity.InitContext) (activity.Activity, error) {
	return &Activity{}, nil
}

func (a *Activity) Metadata() *activity.Metadata {
	return activityMd
}

func (a *Activity) Eval(ctx activity.Context) (bool, error) {
	input := &Input{}
	if err := ctx.GetInputObject(input); err != nil {
		return false, fmt.Errorf("getting activity input: %w", err)
	}

	function, err := validateInput(input)
	if err != nil {
		return false, err
	}

	request := (&Input{IP: input.IP, Port: input.Port, SlaveAddress: input.SlaveAddress,
		RegisterAddress: input.RegisterAddress, Length: input.Length, Function: input.Function}).ToMap()
	value, requestErr := readModbus(input, function)
	var outputErr error
	if requestErr == nil {
		outputErr = ctx.SetOutputObject(&Output{Value: value})
	}

	event := RequestEvent{
		ActivityName: ctx.Name(),
		Request:      request,
		Value:        value,
	}
	if requestErr != nil {
		event.Error = requestErr.Error()
	} else if outputErr != nil {
		event.Error = outputErr.Error()
	}
	if err := PublishRequest(event); err != nil {
		if requestErr != nil {
			return false, fmt.Errorf("%v; publishing Modbus completion event: %w", requestErr, err)
		}
		if outputErr != nil {
			return false, fmt.Errorf("%v; publishing Modbus completion event: %w", outputErr, err)
		}
		return false, fmt.Errorf("publishing Modbus completion event: %w", err)
	}

	if requestErr != nil {
		return false, requestErr
	}
	if outputErr != nil {
		return false, outputErr
	}
	return true, nil
}
