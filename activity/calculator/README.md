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
| Value A  | int32  | First operand - **REQUIRED** |
| Value B  | int32  | Second operand - **REQUIRED** |
| op       | string | Operation: `Sum`, `Sub`, `Mul`, or `Div` |

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
      "Value A": 8,
      "Value B": 2,
      "op": "Sum"
    }
  }
}
```

Supported operation values are `Sum`, `Sub`, `Mul`, and `Div`.