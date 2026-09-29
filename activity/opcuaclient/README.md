# OPC UA Client

This activity connects to an OPC UA server and reads the Value attribute of one node. It does not write values or invoke OPC UA methods.

## Installation

```bash
flogo install github.com/evgen260392/flogo-lib/activity/opcuaclient
```

## Settings

| Name | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| endpoint | string | Yes | Server endpoint, for example `opc.tcp://localhost:4840` |
| nodeID | string | Yes | Node ID in OPC UA string syntax, for example `ns=2;s=Temperature` or `ns=2;i=42` |
| securityPolicy | string | No | Security policy name; defaults to `None` |
| securityMode | string | No | `None`, `Sign`, or `SignAndEncrypt`; defaults to `None` |
| username | string | No | Username for user-token authentication |
| password | string | No | Password for user-token authentication |
| userTokenPolicyID | string | No | User-token policy ID configured by the server |
| timeout | integer | No | Connection/request timeout in seconds; defaults to `10` |

When username authentication is enabled, configure `userTokenPolicyID` if required by the server. Security policy and mode must match a policy supported by the endpoint.

## Output

| Name | Type | Description |
|:-----|:-----|:------------|
| value | any | Value read from the node |

## Example

```json
{
  "id": "read_temperature",
  "name": "Read Temperature",
  "activity": {
    "ref": "github.com/evgen260392/flogo-lib/activity/opcuaclient",
    "settings": {
      "endpoint": "opc.tcp://localhost:4840",
      "nodeID": "ns=2;s=Temperature"
    }
  }
}
```