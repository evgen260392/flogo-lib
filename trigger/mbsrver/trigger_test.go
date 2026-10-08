package mbsrver

import (
	"math"
	"testing"

	"github.com/project-flogo/core/trigger"
)

func TestParseRegisters(t *testing.T) {
	tests := []struct {
		name    string
		entry   map[string]interface{}
		wantLen int
		wantErr bool
	}{
		{
			name: "word",
			entry: map[string]interface{}{
				"name": "word", "address": 10, "type": "Word",
				"mb-function": "ReadHoldingRegisters", "data": 42,
			},
			wantLen: 1,
		},
		{
			name: "float",
			entry: map[string]interface{}{
				"name": "float", "address": 20, "type": "Float",
				"mb-function": "ReadHoldingRegisters", "data": 1.25,
			},
			wantLen: 2,
		},
		{
			name: "string",
			entry: map[string]interface{}{
				"name": "string", "address": 30, "type": "String",
				"mb-function": "ReadHoldingRegisters", "data": "abc",
			},
			wantLen: 2,
		},
		{
			name: "invalid bool register table",
			entry: map[string]interface{}{
				"name": "bool", "address": 10, "type": "Bool",
				"mb-function": "ReadHoldingRegisters", "data": true,
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registers, err := parseRegisters([]interface{}{test.entry})
			if (err != nil) != test.wantErr {
				t.Fatalf("parseRegisters() error = %v, wantErr %v", err, test.wantErr)
			}
			if err == nil && registers[0].length != test.wantLen {
				t.Fatalf("register length = %d, want %d", registers[0].length, test.wantLen)
			}
		})
	}
}

func TestWriteRegisterEmitsOnlyChangedValues(t *testing.T) {
	instance, err := (&Factory{}).New(testConfig([]interface{}{
		map[string]interface{}{
			"name": "temperature", "address": 10, "type": "Word",
			"mb-function": "ReadHoldingRegisters", "data": 42,
		},
	}))
	if err != nil {
		t.Fatalf("creating trigger: %v", err)
	}
	server := instance.(*Trigger).server

	response, events := server.processPDU([]byte{functionWriteSingleRegister, 0, 10, 0, 43})
	if len(response) != 5 || len(events) != 1 || events[0]["data"] != uint16(43) {
		t.Fatalf("write response/events = %v, %#v", response, events)
	}
	if _, events = server.processPDU([]byte{functionWriteSingleRegister, 0, 10, 0, 43}); len(events) != 0 {
		t.Fatalf("writing unchanged value emitted events: %#v", events)
	}
}

func TestWriteFloatEmitsDecodedValue(t *testing.T) {
	instance, err := (&Factory{}).New(testConfig([]interface{}{
		map[string]interface{}{
			"name": "float", "address": 20, "type": "Float",
			"mb-function": "ReadHoldingRegisters", "data": 1.25,
		},
	}))
	if err != nil {
		t.Fatalf("creating trigger: %v", err)
	}
	server := instance.(*Trigger).server

	valueBits := math.Float32bits(2.5)
	pdu := []byte{
		functionWriteMultipleRegs, 0, 20, 0, 2, 4,
		byte(valueBits >> 24), byte(valueBits >> 16), byte(valueBits >> 8), byte(valueBits),
	}
	_, events := server.processPDU(pdu)
	if len(events) != 1 || events[0]["data"] != float32(2.5) {
		t.Fatalf("float write emitted unexpected events: %#v", events)
	}
}

func TestReadUsesConfiguredModbusTable(t *testing.T) {
	instance, err := (&Factory{}).New(testConfig([]interface{}{
		map[string]interface{}{
			"name": "input", "address": 20, "type": "Word",
			"mb-function": "ReadInputRegisters", "data": 1234,
		},
	}))
	if err != nil {
		t.Fatalf("creating trigger: %v", err)
	}
	server := instance.(*Trigger).server

	response, _ := server.processPDU([]byte{functionReadInputRegisters, 0, 20, 0, 1})
	if len(response) != 4 || response[2] != 0x04 || response[3] != 0xd2 {
		t.Fatalf("input-register response = %v, want [4 2 4 210]", response)
	}
	response, _ = server.processPDU([]byte{functionReadHoldingRegisters, 0, 20, 0, 1})
	if len(response) != 2 || response[1] != exceptionIllegalDataAddress {
		t.Fatalf("holding-register response = %v, want illegal data address", response)
	}
}

func TestWriteCoilEmitsBooleanAndReadReturnsIt(t *testing.T) {
	instance, err := (&Factory{}).New(testConfig([]interface{}{
		map[string]interface{}{
			"name": "enabled", "address": 7, "type": "Bool",
			"mb-function": "ReadCoils", "data": false,
		},
	}))
	if err != nil {
		t.Fatalf("creating trigger: %v", err)
	}
	server := instance.(*Trigger).server

	_, events := server.processPDU([]byte{functionWriteSingleCoil, 0, 7, 0xff, 0})
	if len(events) != 1 || events[0]["data"] != true {
		t.Fatalf("coil write emitted unexpected events: %#v", events)
	}
	response, _ := server.processPDU([]byte{functionReadCoils, 0, 7, 0, 1})
	if len(response) != 3 || response[1] != 1 || response[2] != 1 {
		t.Fatalf("coil read response = %v, want [1 1 1]", response)
	}
}

func TestRegisterStorageUsesOnlyConfiguredAddresses(t *testing.T) {
	instance, err := (&Factory{}).New(testConfig([]interface{}{
		map[string]interface{}{
			"name": "low", "address": 3001, "type": "Word",
			"mb-function": "ReadHoldingRegisters", "data": 1,
		},
		map[string]interface{}{
			"name": "high", "address": 65000, "type": "Word",
			"mb-function": "ReadHoldingRegisters", "data": 2,
		},
	}))
	if err != nil {
		t.Fatalf("creating trigger: %v", err)
	}
	server := instance.(*Trigger).server
	if len(server.holding) != 2 {
		t.Fatalf("holding register storage has %d entries, want 2", len(server.holding))
	}
	if server.holding[3001] != 1 || server.holding[65000] != 2 {
		t.Fatalf("configured holding registers were not initialized: %#v", server.holding)
	}
}

func testConfig(registers []interface{}) *trigger.Config {
	return &trigger.Config{
		Id: "test",
		Settings: map[string]interface{}{
			"port":      1502,
			"slaveId":   10,
			"registers": registers,
		},
	}
}
