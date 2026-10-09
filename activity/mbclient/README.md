# Modbus Client

This activity reads coils, discrete inputs, holding registers, or input registers over Modbus TCP. TCP connections are reused for requests with the same server address and slave ID.

## Inputs

| Name | Type | Description |
|:-----|:-----|:------------|
| ip | string | Server IP address or host name |
| port | integer | Server TCP port |
| slaveAddress | integer | Modbus unit ID from 1 to 247 |
| registerAddress | integer | Starting coil or register address |
| length | integer | Number of values to read |
| function | string | `ReadCoils`, `ReadDiscreteInputs`, `ReadHoldingRegisters`, or `ReadInputRegisters` |

The maximum length is 2000 for coil/discrete-input reads and 125 for register reads. Bit reads return `[]bool`; register reads return `[]uint16` in network byte order.

## Output

| Name | Type | Description |
|:-----|:-----|:------------|
| value | any | The decoded values returned by the selected function |

The `trigger/mbclient` trigger emits an event whenever a Modbus request from this activity completes. Successful events include the request and returned value; failed Modbus requests include the request and error message.