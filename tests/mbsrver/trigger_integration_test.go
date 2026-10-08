package mbsrver_test

import (
	"context"
	"encoding/binary"
	"math"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/evgen260392/flogo-lib/trigger/mbsrver"
	"github.com/goburrow/modbus"
	"github.com/project-flogo/core/support/log"
	"github.com/project-flogo/core/trigger"
)

func TestModbusServerTriggerEmitsChangedRegisters(t *testing.T) {
	port := freeTCPPort(t)
	initialString := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWX"
	registers := []interface{}{
		map[string]interface{}{
			"name": "word_3001", "address": 3001, "type": "Word",
			"mb-function": "ReadHoldingRegisters", "data": 11,
		},
		map[string]interface{}{
			"name": "float_3010", "address": 3010, "type": "Float",
			"mb-function": "ReadHoldingRegisters", "data": 1.5,
		},
		map[string]interface{}{
			"name": "word_3100", "address": 3100, "type": "Word",
			"mb-function": "ReadHoldingRegisters", "data": 33,
		},
		map[string]interface{}{
			"name": "word_3101", "address": 3101, "type": "Word",
			"mb-function": "ReadHoldingRegisters", "data": 44,
		},
		map[string]interface{}{
			"name": "string_3200", "address": 3200, "type": "String",
			"mb-function": "ReadHoldingRegisters", "data": initialString,
		},
	}
	handler := &captureHandler{events: make(chan map[string]interface{}, 8)}
	instance, err := (&mbsrver.Factory{}).New(&trigger.Config{
		Id: "mbsrver-integration",
		Settings: map[string]interface{}{
			"port":      port,
			"slaveId":   10,
			"registers": registers,
		},
	})
	if err != nil {
		t.Fatalf("creating Modbus server trigger: %v", err)
	}
	if err := instance.Initialize(testInitContext{handler: handler}); err != nil {
		t.Fatalf("initializing Modbus server trigger: %v", err)
	}
	if err := instance.Start(); err != nil {
		t.Fatalf("starting Modbus server trigger: %v", err)
	}
	defer func() {
		if err := instance.Stop(); err != nil {
			t.Errorf("stopping Modbus server trigger: %v", err)
		}
	}()

	clientHandler := modbus.NewTCPClientHandler(net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	clientHandler.SlaveId = 10
	clientHandler.Timeout = time.Second
	if err := clientHandler.Connect(); err != nil {
		t.Fatalf("connecting Modbus test client: %v", err)
	}
	defer clientHandler.Close()
	client := modbus.NewClient(clientHandler)

	word, err := client.ReadHoldingRegisters(3001, 1)
	if err != nil {
		t.Fatalf("reading initial Word register: %v", err)
	}
	if binary.BigEndian.Uint16(word) != 11 {
		t.Fatalf("initial Word = %d, want 11", binary.BigEndian.Uint16(word))
	}
	float, err := client.ReadHoldingRegisters(3010, 2)
	if err != nil {
		t.Fatalf("reading initial Float register: %v", err)
	}
	floatValue := math.Float32frombits(binary.BigEndian.Uint32(float))
	if floatValue != 1.5 {
		t.Fatalf("initial Float = %v, want 1.5", floatValue)
	}
	stringData, err := client.ReadHoldingRegisters(3200, uint16(len(initialString)/2))
	if err != nil {
		t.Fatalf("reading initial String registers: %v", err)
	}
	if string(stringData) != initialString {
		t.Fatalf("initial String = %q, want %q", string(stringData), initialString)
	}

	if _, err := client.WriteSingleRegister(3001, 12); err != nil {
		t.Fatalf("writing Word register: %v", err)
	}
	assertChangedRegister(t, handler.events, "word_3001", 3001, "Word", "ReadHoldingRegisters", uint16(12))

	updatedFloat := math.Float32bits(2.5)
	floatBytes := []byte{byte(updatedFloat >> 24), byte(updatedFloat >> 16), byte(updatedFloat >> 8), byte(updatedFloat)}
	if _, err := client.WriteMultipleRegisters(3010, 2, floatBytes); err != nil {
		t.Fatalf("writing Float registers: %v", err)
	}
	assertChangedRegister(t, handler.events, "float_3010", 3010, "Float", "ReadHoldingRegisters", float32(2.5))

	if _, err := client.WriteSingleRegister(3001, 12); err != nil {
		t.Fatalf("writing unchanged Word register: %v", err)
	}
	select {
	case event := <-handler.events:
		t.Fatalf("unchanged register unexpectedly emitted an event: %#v", event)
	case <-time.After(200 * time.Millisecond):
	}
}

func assertChangedRegister(t *testing.T, events <-chan map[string]interface{}, name string, address int, dataType, function string, value interface{}) {
	t.Helper()
	select {
	case output := <-events:
		if output["name"] != name {
			t.Errorf("event name = %v, want %q", output["name"], name)
		}
		if output["address"] != address {
			t.Errorf("event address = %v, want %d", output["address"], address)
		}
		if output["type"] != dataType {
			t.Errorf("event type = %v, want %q", output["type"], dataType)
		}
		if output["mb-function"] != function {
			t.Errorf("event mb-function = %v, want %q", output["mb-function"], function)
		}
		if output["data"] != value {
			t.Errorf("event data = %#v, want %#v", output["data"], value)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for changed register %q", name)
	}
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("finding a free TCP port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

type captureHandler struct {
	events chan map[string]interface{}
}

func (*captureHandler) Name() string {
	return "capture"
}

func (*captureHandler) Settings() map[string]interface{} {
	return nil
}

func (*captureHandler) Schemas() *trigger.SchemaConfig {
	return nil
}

func (h *captureHandler) Handle(_ context.Context, value interface{}) (map[string]interface{}, error) {
	output, ok := value.(map[string]interface{})
	if !ok {
		return nil, nil
	}
	event, ok := output["data"].(map[string]interface{})
	if !ok {
		return nil, nil
	}
	h.events <- event
	return nil, nil
}

type testInitContext struct {
	handler trigger.Handler
}

func (c testInitContext) Logger() log.Logger {
	return log.RootLogger()
}

func (c testInitContext) GetHandlers() []trigger.Handler {
	return []trigger.Handler{c.handler}
}
