package mbmaster

import (
	"reflect"
	"testing"
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
