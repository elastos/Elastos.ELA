package errors

import (
	"errors"
)

const callStackDepth = 10

type DetailError interface {
	error
	ErrCoder
	CallStacker
	GetRoot()  error
}


func  NewErr(errmsg string) error {
	return errors.New(errmsg)
}

func NewDetailErr(err error,errcode ErrCode,errmsg string) DetailError{
	if err == nil {return nil}

	elaerr, ok := err.(elaError)
	if !ok {
		elaerr.root = err
		elaerr.errmsg = err.Error()
		elaerr.callstack = getCallStack(0, callStackDepth)
		elaerr.code = errcode

	}
	if errmsg != "" {
		elaerr.errmsg = errmsg + ": " + elaerr.errmsg
	}


	return elaerr
}

func RootErr(err error) error {
	if err, ok := err.(DetailError); ok {
		return err.GetRoot()
	}
	return err
}



