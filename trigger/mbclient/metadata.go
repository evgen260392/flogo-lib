package mbclient

type Settings struct{}

type Output struct {
	ActivityName string                 `md:"activityName"`
	Request      map[string]interface{} `md:"request"`
	Value        interface{}            `md:"value"`
	Error        string                 `md:"error"`
}

func (o *Output) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"activityName": o.ActivityName,
		"request":      o.Request,
		"value":        o.Value,
		"error":        o.Error,
	}
}
