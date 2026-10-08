package mbsrver

type Settings struct {
	Port      int           `md:"port,required"`
	SlaveID   int           `md:"slaveId,required"`
	Registers []interface{} `md:"registers,required"`
}

type Output struct {
	Data interface{} `md:"data"`
}

func (o *Output) ToMap() map[string]interface{} {
	return map[string]interface{}{"data": o.Data}
}
