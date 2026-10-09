package mbclient

import (
	"encoding/binary"
	"io"
	"net"
	"reflect"
	"strconv"
	"sync/atomic"
	"testing"

	flogoactivity "github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/data"
	"github.com/project-flogo/core/support/log"
	"github.com/project-flogo/core/support/trace"
)

func TestDecodeResponse(t *testing.T) {
	tests := []struct {
		name     string
		function readFunction
		response []byte
		length   int
		want     interface{}
	}{
		{
			name:     "coils",
			function: functionReadCoils,
			response: []byte{0x05},
			length:   3,
			want:     []bool{true, false, true},
		},
		{
			name:     "holding registers",
			function: functionReadHoldingRegisters,
			response: []byte{0x12, 0x34, 0xab, 0xcd},
			length:   2,
			want:     []uint16{0x1234, 0xabcd},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := decodeResponse(test.function, test.response, test.length)
			if err != nil {
				t.Fatalf("decodeResponse() error = %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("decodeResponse() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestDecodeResponseRejectsShortPayload(t *testing.T) {
	if _, err := decodeResponse(functionReadInputRegisters, []byte{0x01}, 1); err == nil {
		t.Fatal("decodeResponse() expected an error for a short register response")
	}
}

func TestReadModbusReusesConnection(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("starting fake Modbus server: %v", err)
	}
	defer listener.Close()

	var accepted int32
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		connection, err := listener.Accept()
		if err != nil {
			return
		}
		atomic.AddInt32(&accepted, 1)
		defer connection.Close()
		for {
			header := make([]byte, 7)
			if _, err := io.ReadFull(connection, header); err != nil {
				return
			}
			pduLength := int(binary.BigEndian.Uint16(header[4:6])) - 1
			request := make([]byte, pduLength)
			if _, err := io.ReadFull(connection, request); err != nil {
				return
			}
			response := append(header[:4], 0, 5, header[6], request[0], 2, 0, 42)
			if _, err := connection.Write(response); err != nil {
				return
			}
		}
	}()

	address, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("reading fake Modbus server address: %v", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("parsing fake Modbus server port: %v", err)
	}
	input := &Input{
		IP:              address,
		Port:            port,
		SlaveAddress:    1,
		RegisterAddress: 10,
		Length:          1,
		Function:        string(functionReadHoldingRegisters),
	}
	key := modbusConnectionKey{address: listener.Addr().String(), slaveID: 1}
	cleanupConnection := func() {
		modbusConnections.Lock()
		connection := modbusConnections.items[key]
		modbusConnections.Unlock()
		if connection != nil {
			connection.mu.Lock()
			if !connection.closed {
				connection.handler.Close()
				connection.closed = true
			}
			connection.mu.Unlock()
			removeModbusConnection(key, connection)
		}
	}
	defer cleanupConnection()

	for i := 0; i < 2; i++ {
		value, err := readModbus(input, functionReadHoldingRegisters)
		if err != nil {
			t.Fatalf("readModbus() call %d: %v", i+1, err)
		}
		if !reflect.DeepEqual(value, []uint16{42}) {
			t.Fatalf("readModbus() = %#v, want []uint16{42}", value)
		}
	}

	events, unsubscribe := SubscribeRequests()
	defer unsubscribe()
	ctx := &testActivityContext{
		input: input.ToMap(),
		name:  "read-temperature",
	}
	if done, err := (&Activity{}).Eval(ctx); err != nil || !done {
		t.Fatalf("Activity.Eval() = (%t, %v), want (true, nil)", done, err)
	}
	select {
	case event := <-events:
		if event.ActivityName != "read-temperature" {
			t.Errorf("event activity name = %q, want read-temperature", event.ActivityName)
		}
		if !reflect.DeepEqual(event.Value, []uint16{42}) {
			t.Errorf("event value = %#v, want []uint16{42}", event.Value)
		}
		if event.Error != "" {
			t.Errorf("event error = %q, want empty", event.Error)
		}
	default:
		t.Fatal("Activity.Eval() did not publish its completion event")
	}
	if got := atomic.LoadInt32(&accepted); got != 1 {
		t.Errorf("server accepted %d connections, want 1", got)
	}
	cleanupConnection()
	listener.Close()
	<-serverDone
}

type testActivityContext struct {
	input  map[string]interface{}
	output map[string]interface{}
	name   string
}

func (c *testActivityContext) ActivityHost() flogoactivity.Host {
	return nil
}

func (c *testActivityContext) Name() string {
	return c.name
}

func (c *testActivityContext) GetInput(string) interface{} {
	return nil
}

func (c *testActivityContext) SetOutput(string, interface{}) error {
	return nil
}

func (c *testActivityContext) GetInputObject(input data.StructValue) error {
	return input.FromMap(c.input)
}

func (c *testActivityContext) SetOutputObject(output data.StructValue) error {
	c.output = output.ToMap()
	return nil
}

func (c *testActivityContext) GetSharedTempData() map[string]interface{} {
	return nil
}

func (c *testActivityContext) Logger() log.Logger {
	return log.RootLogger()
}

func (c *testActivityContext) GetTracingContext() trace.TracingContext {
	return nil
}
