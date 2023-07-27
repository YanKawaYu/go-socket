package gotcp

import (
	"errors"
)

//实现了这个接口的错误，panic的时候，就会返回4，同时
type IUserError interface {
	ShowError() string
}

type LibError struct {
	Err error
}

func (e LibError) ShowError() string {
	return e.Err.Error()
}

func raiseError(err string) {
	panic(LibError{errors.New(err)})
}

func getRecoverError(r interface{}) error {
	var err error
	switch x := r.(type) {
	case string:
		err = errors.New(x)
	case error:
		err = x
	default:
		err = errors.New("unknown panic")
	}
	return err
}