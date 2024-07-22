package misc

import "errors"

var (
	ErrArg            = errors.New("incorrect arguments")
	ErrArgSuccessExit = errors.New("arg success exit")
	ErrConfig         = errors.New("config incorrect")
	ErrExecution      = errors.New("execution finished with errors")
)
