module github.com/evgen260392/flogo-lib/tests/mbsrver

go 1.12

require (
	github.com/evgen260392/flogo-lib/trigger/mbsrver v0.0.0
	github.com/goburrow/modbus v0.1.0
	github.com/goburrow/serial v0.1.0 // indirect
	github.com/project-flogo/core v1.1.0
)

replace github.com/evgen260392/flogo-lib/trigger/mbsrver => ../../trigger/mbsrver
