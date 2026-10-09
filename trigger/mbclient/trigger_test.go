package mbclient

import (
	"context"
	"reflect"
	"testing"
	"time"

	activitymbclient "github.com/evgen260392/flogo-lib/activity/mbclient"
	"github.com/project-flogo/core/support/log"
	"github.com/project-flogo/core/trigger"
)

func TestTriggerForwardsActivityCompletion(t *testing.T) {
	handler := &captureHandler{events: make(chan map[string]interface{}, 1)}
	instance, err := (&Factory{}).New(&trigger.Config{Id: "mbclient-test"})
	if err != nil {
		t.Fatalf("creating trigger: %v", err)
	}
	if err := instance.Initialize(testInitContext{handler: handler}); err != nil {
		t.Fatalf("initializing trigger: %v", err)
	}
	if err := instance.Start(); err != nil {
		t.Fatalf("starting trigger: %v", err)
	}
	defer func() {
		if err := instance.Stop(); err != nil {
			t.Errorf("stopping trigger: %v", err)
		}
	}()

	request := map[string]interface{}{"function": "ReadHoldingRegisters"}
	if err := activitymbclient.PublishRequest(activitymbclient.RequestEvent{
		ActivityName: "read-temperature",
		Request:      request,
		Value:        []uint16{215},
	}); err != nil {
		t.Fatalf("publishing request completion: %v", err)
	}

	select {
	case event := <-handler.events:
		if event["activityName"] != "read-temperature" {
			t.Errorf("activityName = %v, want read-temperature", event["activityName"])
		}
		if !reflect.DeepEqual(event["request"], request) {
			t.Errorf("request = %#v, want %#v", event["request"], request)
		}
		if !reflect.DeepEqual(event["value"], []uint16{215}) {
			t.Errorf("event value = %#v, want []uint16{215}", event["value"])
		}
		if event["error"] != "" {
			t.Errorf("event error = %v, want empty", event["error"])
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for request completion event")
	}
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
	event, ok := value.(map[string]interface{})
	if ok {
		h.events <- event
	}
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
