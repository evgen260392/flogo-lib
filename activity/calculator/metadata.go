package calculator

const (
	ivOperandA = "operandA"
	ovResult   = "result"
)

type Settings struct {
	OperandB interface{} `md:"Operand B,required"`
	Operator string      `md:"operator,allowed(Add,Sub,Mul,Div)"`
}

type Input struct {
	OperandA interface{} `md:"operandA,required"`
}

func (i *Input) ToMap() map[string]interface{} {
	return map[string]interface{}{ivOperandA: i.OperandA}
}

func (i *Input) FromMap(values map[string]interface{}) error {
	i.OperandA = values[ivOperandA]
	return nil
}

type Output struct {
	Result interface{} `md:"result"`
}

func (o *Output) ToMap() map[string]interface{} {
	return map[string]interface{}{ovResult: o.Result}
}

func (o *Output) FromMap(values map[string]interface{}) error {
	o.Result = values[ovResult]
	return nil
}
