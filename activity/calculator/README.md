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
| Operand B | any    | Second operand; any numeric type - **REQUIRED** |
| operator  | string | Operation: `Add`, `Sub`, `Mul`, or `Div` |

### Input:
| Name   | Type | Description |
|:-------|:-----|:------------|
| operandA | any  | First operand from the flow; any numeric type - **REQUIRED** |

### Output:
| Name  | Type | Description |
|:------|:-----|:------------|
| result | int  | Result of the arithmetic operation |

## Examples

### Add
Add two values:

```json
{
  "id": "sum_values",
  "name": "Add Values",
  "activity": {
    "ref": "github.com/evgen260392/flogo-lib/activity/calculator",
    "settings": {
      "Operand B": 2,
      "operator": "Add"
    },
    "input": {
      "operandA": "=$flow.value"
    }
  }
}
```

Supported operation values are `Add`, `Sub`, `Mul`, and `Div`.