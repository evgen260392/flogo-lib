package mbpoller

type Settings struct {
	Host     string `md:"host,required"`
	Port     int    `md:"port,required"`
	Interval int    `md:"interval,required"`
}

type HandlerSettings struct {
	SlaveAddress int                    `md:"slaveAddress,required"`
	Registers    map[string]interface{} `md:"registers,required"`
}

type Output struct {
	Data []interface{} `md:"data"`
}

func (o *Output) ToMap() map[string]interface{} {
	return map[string]interface{}{"data": o.Data}
}
