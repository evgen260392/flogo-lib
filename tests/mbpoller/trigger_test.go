package mbpoller_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/evgen260392/flogo-lib/trigger/mbpoller"
	"github.com/project-flogo/core/support/log"
	"github.com/project-flogo/core/trigger"
)

const (
	modbusAddress = "127.0.0.1:502"
	slaveAddress  = 10
)

func TestTriggerPollsConfiguredRegistersFromLocalServer(t *testing.T) {
	connection, err := net.DialTimeout("tcp", modbusAddress, time.Second)
	if err != nil {
		t.Skipf("Modbus TCP integration server is unavailable at %s: %v", modbusAddress, err)
	}
	_ = connection.Close()

	registers := map[string]interface{}{
		"word_3001":   3001,
		"float_3010":  3010,
		"word_3100":   3100,
		"word_3101":   3101,
		"string_3200": 3200,
	}
	expected := []struct {
		name    string
		address int
	}{
		{name: "float_3010", address: 3010},
		{name: "string_3200", address: 3200},
		{name: "word_3001", address: 3001},
		{name: "word_3100", address: 3100},
		{name: "word_3101", address: 3101},
	}
	handler := &captureHandler{
		settings: map[string]interface{}{
			"slaveAddress": slaveAddress,
			"registers":    registers,
		},
		events: make(chan interface{}, 4),
	}
	factory := &mbpoller.Factory{}
	instance, err := factory.New(&trigger.Config{
		Id: "mbpoller-integration",
		Settings: map[string]interface{}{
			"host":     "127.0.0.1",
			"port":     502,
			"interval": 500,
		},
	})
	if err != nil {
		t.Fatalf("creating Modbus poller trigger: %v", err)
	}
	if err := instance.Initialize(testInitContext{handler: handler}); err != nil {
		t.Fatalf("initializing Modbus poller trigger: %v", err)
	}
	if err := instance.Start(); err != nil {
		t.Fatalf("starting Modbus poller trigger: %v", err)
	}
	defer func() {
		if err := instance.Stop(); err != nil {
			t.Errorf("stopping Modbus poller trigger: %v", err)
		}
	}()

	for poll := 1; poll <= 2; poll++ {
		select {
		case event := <-handler.events:
			assertRegisterOutput(t, event, expected)
		case <-time.After(15 * time.Second):
			t.Fatalf("timed out waiting for Modbus poll %d", poll)
		}
	}
}

func assertRegisterOutput(t *testing.T, event interface{}, expected []struct {
	name    string
	address int
}) {
	t.Helper()

	output, ok := event.(map[string]interface{})
	if !ok {
		t.Fatalf("trigger output has type %T, want map[string]interface{}", event)
	}
	data, ok := output["data"].([]interface{})
	if !ok {
		t.Fatalf("trigger output field data has type %T, want []interface{}", output["data"])
	}
	if len(data) != len(expected) {
		t.Fatalf("trigger returned %d registers, want %d", len(data), len(expected))
	}

	for index, expectedRegister := range expected {
		item, ok := data[index].(map[string]interface{})
		if !ok {
			t.Fatalf("register %d has type %T, want map[string]interface{}", index, data[index])
		}
		if item["name"] != expectedRegister.name {
			t.Errorf("register %d name = %v, want %q", index, item["name"], expectedRegister.name)
		}
		if item["address"] != expectedRegister.address {
			t.Errorf("register %q address = %v, want %d", expectedRegister.name, item["address"], expectedRegister.address)
		}
		if _, ok := item["value"].(uint16); !ok {
			t.Errorf("register %q value has type %T, want uint16", expectedRegister.name, item["value"])
		}
	}
}

type captureHandler struct {
	settings map[string]interface{}
	events   chan interface{}
}

func (h *captureHandler) Name() string {
	return "modbus-server"
}

func (h *captureHandler) Settings() map[string]interface{} {
	return h.settings
}

func (*captureHandler) Schemas() *trigger.SchemaConfig {
	return nil
}

func (h *captureHandler) Handle(_ context.Context, event interface{}) (map[string]interface{}, error) {
	select {
	case h.events <- event:
		return nil, nil
	default:
		return nil, fmt.Errorf("integration test event buffer is full")
	}
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
