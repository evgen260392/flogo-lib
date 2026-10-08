package mbpoller

import (
	"context"
	"testing"

	"github.com/project-flogo/core/data/metadata"
	"github.com/project-flogo/core/support"
	"github.com/project-flogo/core/trigger"
)

type testHandler struct{}

func (testHandler) Name() string {
	return "test"
}

func (testHandler) Settings() map[string]interface{} {
	return nil
}

func (testHandler) Schemas() *trigger.SchemaConfig {
	return nil
}

func (testHandler) Handle(context.Context, interface{}) (map[string]interface{}, error) {
	return nil, nil
}

func TestTriggerRegister(t *testing.T) {
	ref := support.GetRef(&Trigger{})
	if factory := trigger.GetFactory(ref); factory == nil {
		t.Fatal("expected trigger factory to be registered")
	}
}

func TestValidateSettings(t *testing.T) {
	tests := []struct {
		name     string
		settings Settings
		wantErr  bool
	}{
		{
			name:     "valid",
			settings: Settings{Host: "localhost", Port: 502, Interval: 1000},
		},
		{
			name:     "missing host",
			settings: Settings{Port: 502, Interval: 1000},
			wantErr:  true,
		},
		{
			name:     "invalid port",
			settings: Settings{Host: "localhost", Port: 65536, Interval: 1000},
			wantErr:  true,
		},
		{
			name:     "invalid interval",
			settings: Settings{Host: "localhost", Port: 502, Interval: 0},
			wantErr:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateSettings(&test.settings)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateSettings() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestNewPollHandler(t *testing.T) {
	tests := []struct {
		name     string
		settings HandlerSettings
		wantErr  bool
	}{
		{
			name: "valid register map",
			settings: HandlerSettings{
				SlaveAddress: 1,
				Registers:    map[string]interface{}{"temperature": float64(12), "pressure": 0},
			},
		},
		{
			name: "invalid slave address",
			settings: HandlerSettings{
				SlaveAddress: 248,
				Registers:    map[string]interface{}{"temperature": 12},
			},
			wantErr: true,
		},
		{
			name: "register address out of range",
			settings: HandlerSettings{
				SlaveAddress: 1,
				Registers:    map[string]interface{}{"temperature": 65536},
			},
			wantErr: true,
		},
		{
			name: "empty register map",
			settings: HandlerSettings{
				SlaveAddress: 1,
				Registers:    map[string]interface{}{},
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := newPollHandler(&test.settings, testHandler{})
			if (err != nil) != test.wantErr {
				t.Fatalf("newPollHandler() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestHandlerSettingsMapToStruct(t *testing.T) {
	settings := &HandlerSettings{}
	err := metadata.MapToStruct(map[string]interface{}{
		"slaveAddress": 1,
		"registers": map[string]interface{}{
			"temperature": float64(12),
		},
	}, settings, true)
	if err != nil {
		t.Fatalf("MapToStruct() error = %v", err)
	}

	poller, err := newPollHandler(settings, testHandler{})
	if err != nil {
		t.Fatalf("newPollHandler() error = %v", err)
	}
	if len(poller.registers) != 1 || poller.registers[0].name != "temperature" || poller.registers[0].address != 12 {
		t.Fatalf("unexpected parsed register map: %#v", poller.registers)
	}
}
