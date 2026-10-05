package mbmaster

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/goburrow/modbus"
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

	address := net.JoinHostPort(input.IP, strconv.Itoa(input.Port))
	handler := modbus.NewTCPClientHandler(address)
	handler.SlaveId = byte(input.SlaveAddress)
	handler.Timeout = 10 * time.Second
	if err := handler.Connect(); err != nil {
		return false, fmt.Errorf("connecting to Modbus server %q: %w", address, err)
	}
	defer handler.Close()

	client := modbus.NewClient(handler)
	var response []byte
	switch function {
	case functionReadCoils:
		response, err = client.ReadCoils(uint16(input.RegisterAddress), uint16(input.Length))
	case functionReadDiscreteInputs:
		response, err = client.ReadDiscreteInputs(uint16(input.RegisterAddress), uint16(input.Length))
	case functionReadHoldingRegisters:
		response, err = client.ReadHoldingRegisters(uint16(input.RegisterAddress), uint16(input.Length))
	case functionReadInputRegisters:
		response, err = client.ReadInputRegisters(uint16(input.RegisterAddress), uint16(input.Length))
	}
	if err != nil {
		return false, fmt.Errorf("Modbus %s at address %d: %w", function, input.RegisterAddress, err)
	}

	value, err := decodeResponse(function, response, input.Length)
	if err != nil {
		return false, err
	}
	if err := ctx.SetOutputObject(&Output{Value: value}); err != nil {
		return false, err
	}

	return true, nil
}

type readFunction string

const (
	functionReadCoils            readFunction = "ReadCoils"
	functionReadDiscreteInputs   readFunction = "ReadDiscreteInputs"
	functionReadHoldingRegisters readFunction = "ReadHoldingRegisters"
	functionReadInputRegisters   readFunction = "ReadInputRegisters"
)

func validateInput(input *Input) (readFunction, error) {
	if input.IP == "" {
		return "", fmt.Errorf("input %q is required", ivIP)
	}
	if input.Port < 1 || input.Port > 65535 {
		return "", fmt.Errorf("input %q must be between 1 and 65535", ivPort)
	}
	if input.SlaveAddress < 1 || input.SlaveAddress > 247 {
		return "", fmt.Errorf("input %q must be between 1 and 247", ivSlaveAddress)
	}
	if input.RegisterAddress < 0 || input.RegisterAddress > 65535 {
		return "", fmt.Errorf("input %q must be between 0 and 65535", ivRegisterAddress)
	}
	if input.Length < 1 {
		return "", fmt.Errorf("input %q must be greater than zero", ivLength)
	}
	if input.RegisterAddress+input.Length > 65536 {
		return "", fmt.Errorf("input %q and %q exceed the Modbus address range", ivRegisterAddress, ivLength)
	}

	function := readFunction(input.Function)
	maxLength := 125
	switch function {
	case functionReadCoils, functionReadDiscreteInputs:
		maxLength = 2000
	case functionReadHoldingRegisters, functionReadInputRegisters:
	default:
		return "", fmt.Errorf("input %q must be one of ReadCoils, ReadDiscreteInputs, ReadHoldingRegisters, ReadInputRegisters", ivFunction)
	}
	if input.Length > maxLength {
		return "", fmt.Errorf("input %q must not exceed %d for %s", ivLength, maxLength, function)
	}

	return function, nil
}

func decodeResponse(function readFunction, response []byte, length int) (interface{}, error) {
	switch function {
	case functionReadCoils, functionReadDiscreteInputs:
		expectedBytes := (length + 7) / 8
		if len(response) < expectedBytes {
			return nil, fmt.Errorf("Modbus response has %d bytes; expected at least %d", len(response), expectedBytes)
		}
		values := make([]bool, length)
		for i := range values {
			values[i] = response[i/8]&(1<<uint(i%8)) != 0
		}
		return values, nil
	case functionReadHoldingRegisters, functionReadInputRegisters:
		expectedBytes := length * 2
		if len(response) < expectedBytes {
			return nil, fmt.Errorf("Modbus response has %d bytes; expected at least %d", len(response), expectedBytes)
		}
		values := make([]uint16, length)
		for i := range values {
			values[i] = binary.BigEndian.Uint16(response[i*2 : i*2+2])
		}
		return values, nil
	default:
		return nil, fmt.Errorf("unsupported Modbus function %q", function)
	}
}
