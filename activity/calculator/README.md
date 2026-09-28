# Calculator
This activity performs arithmetic operations on two values.

## Installation

### Flogo CLI
```bash
flogo install github.com/evgen260392/flogo-lib/activity/calculator
```

## Configuration

### Settings:
| Name     | Type   | Description |
|:---------|:-------|:------------|
| Value B  | any    | Second operand; any numeric type - **REQUIRED** |
| op       | string | Operation: `Sum`, `Sub`, `Mul`, or `Div` |

### Input:
| Name   | Type | Description |
|:-------|:-----|:------------|
| valueA | any  | First operand from the flow; any numeric type - **REQUIRED** |

### Output:
| Name  | Type | Description |
|:------|:-----|:------------|
| value | int  | Result of the arithmetic operation |

## Examples

### Sum
Add two values:

```json
{
  "id": "sum_values",
  "name": "Sum Values",
  "activity": {
    "ref": "github.com/evgen260392/flogo-lib/activity/calculator",
    "settings": {
      "Value B": 2,
      "op": "Sum"
    },
    "input": {
      "valueA": "=$flow.value"
    }
  }
}
```

Supported operation values are `Sum`, `Sub`, `Mul`, and `Div`.