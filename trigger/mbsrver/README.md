# Modbus TCP Server Trigger

The trigger starts a Modbus TCP server bound to `127.0.0.1` and emits an event
when a Modbus client writes a value that differs from the current value.
Register values are stored only for addresses present in the configured map;
unconfigured addresses are not allocated and return an illegal-data-address
exception when requested.

## Settings

| Name | Type | Description |
|:-----|:-----|:------------|
| `port` | integer | Local TCP port |
| `slaveId` | integer | Modbus Unit ID, from 1 to 247 |
| `registers` | array | Register definitions, including initial data |

Each `registers` item is an object:

```json
{
  "name": "temperature",
  "address": 3001,
  "type": "Word",
  "mb-function": "ReadHoldingRegisters",
  "data": 215
}
```

Supported types are `Word` (one 16-bit register), `Float` (two consecutive
registers, IEEE-754 float32, most-significant word first), `String` (UTF-8
bytes in consecutive registers, with capacity determined by the initial string
length), and `Bool` (one bit).

Supported `mb-function` values identify the Modbus table:

* `ReadHoldingRegisters` / `WriteSingleRegister` / `WriteMultipleRegisters`
* `ReadInputRegisters`
* `ReadCoils` / `WriteSingleCoil` / `WriteMultipleCoils`
* `ReadDiscreteInputs`

Holding registers and coils accept writes. Input registers and discrete inputs
are read-only. Word/float/string types require a 16-bit register table; `Bool`
requires a bit table. Read requests use the corresponding function codes;
holding registers and coils can also be written using their standard write
function codes.

On a successful write that changes a configured value, the handler receives one
event with `data` set to the changed register object:

```json
{
  "data": {
    "name": "temperature",
    "address": 3001,
    "type": "Word",
    "mb-function": "ReadHoldingRegisters",
    "data": 216
  }
}
```

The server only accepts TCP connections through the loopback interface and
ignores requests for other Unit IDs.
