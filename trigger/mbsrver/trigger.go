package mbsrver

import (
	"fmt"

	"github.com/project-flogo/core/data/metadata"
	"github.com/project-flogo/core/trigger"
)

var triggerMd = trigger.NewMetadata(&Settings{}, &Output{})

func init() {
	_ = trigger.Register(&Trigger{}, &Factory{})
}

type Trigger struct {
	server *modbusServer
}

type Factory struct{}

func (*Factory) New(config *trigger.Config) (trigger.Trigger, error) {
	if config == nil {
		return nil, fmt.Errorf("trigger config is required")
	}
	settings := &Settings{}
	if err := metadata.MapToStruct(config.Settings, settings, true); err != nil {
		return nil, err
	}
	server, err := newModbusServer(config.Id, settings)
	if err != nil {
		return nil, err
	}
	return &Trigger{server: server}, nil
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
		return fmt.Errorf("Modbus server requires at least one handler")
	}
	t.server.initialize(ctx.Logger(), handlers)
	return nil
}

func (t *Trigger) Start() error {
	return t.server.start()
}

func (t *Trigger) Stop() error {
	return t.server.stop()
}
