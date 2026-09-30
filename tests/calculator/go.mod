module github.com/evgen260392/flogo-lib/tests/calculator

go 1.12

require (
	github.com/evgen260392/flogo-lib/activity/calculator v0.0.0
	github.com/project-flogo/core v1.1.0
	github.com/stretchr/testify v1.12.1
)

replace github.com/evgen260392/flogo-lib/activity/calculator => ../../activity/calculator
