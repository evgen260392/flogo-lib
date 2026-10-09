package mbclient

import (
	"context"
	"fmt"
	"sync"

	activitymbclient "github.com/evgen260392/flogo-lib/activity/mbclient"
	"github.com/project-flogo/core/support/log"
	"github.com/project-flogo/core/trigger"
)

var triggerMd = trigger.NewMetadata(&Settings{}, &Output{})

func init() {
	_ = trigger.Register(&Trigger{}, &Factory{})
}

type Trigger struct {
	mu          sync.Mutex
	logger      log.Logger
	handlers    []trigger.Handler
	events      <-chan activitymbclient.RequestEvent
	unsubscribe func()
	started     bool
	wg          sync.WaitGroup
}

type Factory struct{}

func (*Factory) New(config *trigger.Config) (trigger.Trigger, error) {
	if config == nil {
		return nil, fmt.Errorf("trigger config is required")
	}
	return &Trigger{}, nil
}

func (*Factory) Metadata() *trigger.Metadata {
	return triggerMd
}

func (*Trigger) Metadata() *trigger.Metadata {
	return triggerMd
}

func (t *Trigger) Initialize(ctx trigger.InitContext) error {
	handlers := ctx.GetHandlers()
	if len(handlers) == 0 {
		return fmt.Errorf("Modbus client trigger requires at least one handler")
	}
	t.mu.Lock()
	t.logger = ctx.Logger()
	t.handlers = handlers
	t.mu.Unlock()
	return nil
}

func (t *Trigger) Start() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.started {
		return fmt.Errorf("Modbus client trigger is already started")
	}
	if len(t.handlers) == 0 {
		return fmt.Errorf("Modbus client trigger is not initialized")
	}
	t.events, t.unsubscribe = activitymbclient.SubscribeRequests()
	t.started = true
	t.wg.Add(1)
	go t.listen(t.events)
	return nil
}

func (t *Trigger) Stop() error {
	t.mu.Lock()
	if !t.started {
		t.mu.Unlock()
		return nil
	}
	unsubscribe := t.unsubscribe
	t.started = false
	t.mu.Unlock()

	unsubscribe()
	t.wg.Wait()
	return nil
}

func (t *Trigger) listen(events <-chan activitymbclient.RequestEvent) {
	defer t.wg.Done()
	for event := range events {
		output := (&Output{
			ActivityName: event.ActivityName,
			Request:      event.Request,
			Value:        event.Value,
			Error:        event.Error,
		}).ToMap()
		for _, handler := range t.handlers {
			if _, err := handler.Handle(context.Background(), output); err != nil {
				t.logger.Errorf("Modbus client trigger handler %q failed: %v", handler.Name(), err)
			}
		}
	}
}
