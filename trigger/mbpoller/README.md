# Modbus TCP Poller Trigger

This trigger periodically reads configured holding registers from Modbus TCP
slaves. The trigger supports Modbus TCP only.

## Trigger settings

| Name | Type | Description |
|:-----|:-----|:------------|
| `host` | string | Modbus TCP server IP address or host name |
| `port` | integer | Modbus TCP server port |
| `interval` | integer | Polling interval in milliseconds |

## Handler settings

Each handler polls one slave. Configure its unit ID and a map of register names
to zero-based holding-register addresses. For example:

```json
{
  "slaveAddress": 1,
  "registers": {
    "temperature": 0,
    "pressure": 12
  }
}
```

The trigger reads each configured register as one unsigned 16-bit holding
register.

## Output

The handler receives a `data` array. Each item contains the configured register
name, its address, and the value read from the slave:

```json
{
  "data": [
    { "name": "temperature", "address": 0, "value": 215 },
    { "name": "pressure", "address": 12, "value": 980 }
  ]
}
```

Polling starts immediately when the trigger starts and repeats at the configured
interval. Connection or read errors are logged, and polling continues on the
next interval.
