package mbpoller

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/goburrow/modbus"
	"github.com/project-flogo/core/data/coerce"
	"github.com/project-flogo/core/data/metadata"
	"github.com/project-flogo/core/support/log"
	"github.com/project-flogo/core/trigger"
)

var triggerMd = trigger.NewMetadata(&Settings{}, &HandlerSettings{}, &Output{})

func init() {
	_ = trigger.Register(&Trigger{}, &Factory{})
}

type Trigger struct {
	settings *Settings
	id       string

	mu       sync.Mutex
	cancel   context.CancelFunc
	wait     sync.WaitGroup
	handlers []pollHandler
	logger   log.Logger
}

type Factory struct{}

func (*Factory) New(config *trigger.Config) (trigger.Trigger, error) {
	settings := &Settings{}
	if err := metadata.MapToStruct(config.Settings, settings, true); err != nil {
		return nil, err
	}
	if err := validateSettings(settings); err != nil {
		return nil, err
	}

	return &Trigger{id: config.Id, settings: settings}, nil
}

func (*Factory) Metadata() *trigger.Metadata {
	return triggerMd
}

func (*Trigger) Metadata() *trigger.Metadata {
	return triggerMd
}

func (t *Trigger) Initialize(ctx trigger.InitContext) error {
	t.logger = ctx.Logger()

	handlers := ctx.GetHandlers()
	if len(handlers) == 0 {
		return fmt.Errorf("Modbus poller requires at least one handler")
	}

	for _, handler := range handlers {
		settings := &HandlerSettings{}
		if err := metadata.MapToStruct(handler.Settings(), settings, true); err != nil {
			return fmt.Errorf("handler %q settings: %w", handler.Name(), err)
		}
		poller, err := newPollHandler(settings, handler)
		if err != nil {
			return fmt.Errorf("handler %q settings: %w", handler.Name(), err)
		}
		t.handlers = append(t.handlers, poller)
	}

	return nil
}

func (t *Trigger) Start() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cancel != nil {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	for _, handler := range t.handlers {
		t.wait.Add(1)
		go t.poll(ctx, handler)
	}
	return nil
}

func (t *Trigger) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cancel == nil {
		return nil
	}
	t.cancel()
	t.wait.Wait()
	t.cancel = nil
	return nil
}

func (t *Trigger) poll(ctx context.Context, handler pollHandler) {
	defer t.wait.Done()

	select {
	case <-ctx.Done():
		return
	default:
	}
	t.pollOnce(ctx, handler)
	ticker := time.NewTicker(time.Duration(t.settings.Interval) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.pollOnce(ctx, handler)
		}
	}
}

func (t *Trigger) pollOnce(ctx context.Context, handler pollHandler) {
	data, err := t.readRegisters(handler)
	if err != nil {
		t.logger.Errorf("Modbus poll failed for trigger %q handler %q: %v", t.id, handler.name, err)
		return
	}
	if _, err := handler.handler.Handle(ctx, (&Output{Data: data}).ToMap()); err != nil {
		t.logger.Errorf("Modbus handler %q failed: %v", handler.name, err)
	}
}

func (t *Trigger) readRegisters(handler pollHandler) ([]interface{}, error) {
	address := net.JoinHostPort(t.settings.Host, strconv.Itoa(t.settings.Port))
	clientHandler := modbus.NewTCPClientHandler(address)
	clientHandler.SlaveId = byte(handler.slaveAddress)
	clientHandler.Timeout = 5 * time.Second
	if err := clientHandler.Connect(); err != nil {
		return nil, fmt.Errorf("connecting to Modbus server %q: %w", address, err)
	}
	defer clientHandler.Close()

	client := modbus.NewClient(clientHandler)
	data := make([]interface{}, 0, len(handler.registers))
	for _, register := range handler.registers {
		response, err := client.ReadHoldingRegisters(uint16(register.address), 1)
		if err != nil {
			return nil, fmt.Errorf("reading register %q at address %d from slave %d: %w",
				register.name, register.address, handler.slaveAddress, err)
		}
		if len(response) < 2 {
			return nil, fmt.Errorf("Modbus response for register %q has %d bytes; expected at least 2",
				register.name, len(response))
		}
		data = append(data, map[string]interface{}{
			"name":    register.name,
			"address": register.address,
			"value":   binary.BigEndian.Uint16(response[:2]),
		})
	}
	return data, nil
}

type pollHandler struct {
	name         string
	slaveAddress int
	registers    []register
	handler      trigger.Handler
}

type register struct {
	name    string
	address int
}

func newPollHandler(settings *HandlerSettings, handler trigger.Handler) (pollHandler, error) {
	if settings.SlaveAddress < 1 || settings.SlaveAddress > 247 {
		return pollHandler{}, fmt.Errorf("setting %q must be between 1 and 247", "slaveAddress")
	}
	if len(settings.Registers) == 0 {
		return pollHandler{}, fmt.Errorf("setting %q must contain at least one register", "registers")
	}

	names := make([]string, 0, len(settings.Registers))
	for name := range settings.Registers {
		names = append(names, name)
	}
	sort.Strings(names)

	registers := make([]register, 0, len(names))
	for _, name := range names {
		if name == "" {
			return pollHandler{}, fmt.Errorf("register names must not be empty")
		}
		address, err := coerce.ToInt(settings.Registers[name])
		if err != nil {
			return pollHandler{}, fmt.Errorf("register %q address: %w", name, err)
		}
		if address < 0 || address > 65535 {
			return pollHandler{}, fmt.Errorf("register %q address must be between 0 and 65535", name)
		}
		registers = append(registers, register{name: name, address: address})
	}

	return pollHandler{
		name:         handler.Name(),
		slaveAddress: settings.SlaveAddress,
		registers:    registers,
		handler:      handler,
	}, nil
}

func validateSettings(settings *Settings) error {
	if settings.Host == "" {
		return fmt.Errorf("setting %q is required", "host")
	}
	if settings.Port < 1 || settings.Port > 65535 {
		return fmt.Errorf("setting %q must be between 1 and 65535", "port")
	}
	if settings.Interval < 1 {
		return fmt.Errorf("setting %q must be at least 1 millisecond", "interval")
	}
	if time.Duration(settings.Interval) > time.Duration(1<<63-1)/time.Millisecond {
		return fmt.Errorf("setting %q is too large", "interval")
	}
	return nil
}
