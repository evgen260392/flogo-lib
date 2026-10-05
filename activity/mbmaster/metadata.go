package mbmaster

const (
	ivIP              = "ip"
	ivPort            = "port"
	ivSlaveAddress    = "slaveAddress"
	ivRegisterAddress = "registerAddress"
	ivLength          = "length"
	ivFunction        = "function"
	ovValue           = "value"
)

type Input struct {
	IP              string `md:"ip,required"`
	Port            int    `md:"port,required"`
	SlaveAddress    int    `md:"slaveAddress,required"`
	RegisterAddress int    `md:"registerAddress,required"`
	Length          int    `md:"length,required"`
	Function        string `md:"function,required,allowed(ReadCoils,ReadDiscreteInputs,ReadHoldingRegisters,ReadInputRegisters)"`
}

func (i *Input) ToMap() map[string]interface{} {
	return map[string]interface{}{
		ivIP:              i.IP,
		ivPort:            i.Port,
		ivSlaveAddress:    i.SlaveAddress,
		ivRegisterAddress: i.RegisterAddress,
		ivLength:          i.Length,
		ivFunction:        i.Function,
	}
}

func (i *Input) FromMap(values map[string]interface{}) error {
	i.IP, _ = values[ivIP].(string)
	i.Port, _ = values[ivPort].(int)
	i.SlaveAddress, _ = values[ivSlaveAddress].(int)
	i.RegisterAddress, _ = values[ivRegisterAddress].(int)
	i.Length, _ = values[ivLength].(int)
	i.Function, _ = values[ivFunction].(string)
	return nil
}

type Output struct {
	Value interface{} `md:"value"`
}

func (o *Output) ToMap() map[string]interface{} {
	return map[string]interface{}{ovValue: o.Value}
}

func (o *Output) FromMap(values map[string]interface{}) error {
	o.Value = values[ovValue]
	return nil
}
