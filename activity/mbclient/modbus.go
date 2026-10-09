package mbclient

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/goburrow/modbus"
)

type readFunction string

const (
	functionReadCoils            readFunction = "ReadCoils"
	functionReadDiscreteInputs   readFunction = "ReadDiscreteInputs"
	functionReadHoldingRegisters readFunction = "ReadHoldingRegisters"
	functionReadInputRegisters   readFunction = "ReadInputRegisters"
)

type RequestEvent struct {
	ActivityName string
	Request      map[string]interface{}
	Value        interface{}
	Error        string
}

type requestSubscription struct {
	events chan RequestEvent
}

var requestSubscriptions = struct {
	sync.Mutex
	nextID uint64
	items  map[uint64]*requestSubscription
}{items: make(map[uint64]*requestSubscription)}

// SubscribeRequests listens for completed Modbus requests made by this activity.
func SubscribeRequests() (<-chan RequestEvent, func()) {
	requestSubscriptions.Lock()
	requestSubscriptions.nextID++
	id := requestSubscriptions.nextID
	subscription := &requestSubscription{events: make(chan RequestEvent, 100)}
	requestSubscriptions.items[id] = subscription
	requestSubscriptions.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			requestSubscriptions.Lock()
			delete(requestSubscriptions.items, id)
			close(subscription.events)
			requestSubscriptions.Unlock()
		})
	}
	return subscription.events, unsubscribe
}

// PublishRequest notifies all active Modbus completion trigger subscriptions.
func PublishRequest(event RequestEvent) error {
	requestSubscriptions.Lock()
	defer requestSubscriptions.Unlock()

	var full int
	for _, subscription := range requestSubscriptions.items {
		select {
		case subscription.events <- event:
		default:
			full++
		}
	}
	if full != 0 {
		return fmt.Errorf("Modbus completion event queue is full for %d subscriber(s)", full)
	}
	return nil
}

type modbusConnectionKey struct {
	address string
	slaveID byte
}

type modbusConnection struct {
	mu      sync.Mutex
	handler *modbus.TCPClientHandler
	client  modbus.Client
	closed  bool
}

var modbusConnections = struct {
	sync.Mutex
	items map[modbusConnectionKey]*modbusConnection
}{items: make(map[modbusConnectionKey]*modbusConnection)}

func readModbus(input *Input, function readFunction) (interface{}, error) {
	address := net.JoinHostPort(input.IP, strconv.Itoa(input.Port))
	key := modbusConnectionKey{address: address, slaveID: byte(input.SlaveAddress)}

	for {
		connection, err := getModbusConnection(key)
		if err != nil {
			return nil, fmt.Errorf("connecting to Modbus server %q: %w", address, err)
		}
		connection.mu.Lock()
		if connection.closed {
			connection.mu.Unlock()
			continue
		}

		response, err := readFromModbus(connection.client, function, input)
		if err != nil {
			connection.handler.Close()
			connection.closed = true
			removeModbusConnection(key, connection)
			connection.mu.Unlock()
			return nil, fmt.Errorf("Modbus %s at address %d: %w", function, input.RegisterAddress, err)
		}
		connection.mu.Unlock()

		value, err := decodeResponse(function, response, input.Length)
		if err != nil {
			return nil, err
		}
		return value, nil
	}
}

func getModbusConnection(key modbusConnectionKey) (*modbusConnection, error) {
	modbusConnections.Lock()
	defer modbusConnections.Unlock()

	if connection := modbusConnections.items[key]; connection != nil {
		return connection, nil
	}

	handler := modbus.NewTCPClientHandler(key.address)
	handler.SlaveId = key.slaveID
	handler.Timeout = 10 * time.Second
	if err := handler.Connect(); err != nil {
		return nil, err
	}
	connection := &modbusConnection{
		handler: handler,
		client:  modbus.NewClient(handler),
	}
	modbusConnections.items[key] = connection
	return connection, nil
}

func removeModbusConnection(key modbusConnectionKey, connection *modbusConnection) {
	modbusConnections.Lock()
	if modbusConnections.items[key] == connection {
		delete(modbusConnections.items, key)
	}
	modbusConnections.Unlock()
}

func readFromModbus(client modbus.Client, function readFunction, input *Input) ([]byte, error) {
	switch function {
	case functionReadCoils:
		return client.ReadCoils(uint16(input.RegisterAddress), uint16(input.Length))
	case functionReadDiscreteInputs:
		return client.ReadDiscreteInputs(uint16(input.RegisterAddress), uint16(input.Length))
	case functionReadHoldingRegisters:
		return client.ReadHoldingRegisters(uint16(input.RegisterAddress), uint16(input.Length))
	case functionReadInputRegisters:
		return client.ReadInputRegisters(uint16(input.RegisterAddress), uint16(input.Length))
	default:
		return nil, fmt.Errorf("unsupported Modbus function %q", function)
	}
}

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
