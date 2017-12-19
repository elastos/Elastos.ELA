package errors

type elaError struct {
	errmsg string
	callstack *CallStack
	root error
	code ErrCode
}

func (e elaError) Error() string {
	return e.errmsg
}

func (e elaError) GetErrCode()  ErrCode {
	return e.code
}

func (e elaError) GetRoot()  error {
	return e.root
}

func (e elaError) GetCallStack()  *CallStack {
	return e.callstack
}
