package gosocket

import (
	"errors"
)

// IUserError
// Whenever you panic an error that implements this interface in the action, the response message will be the return result of ShowError
// For example, {"status":4, "message":"this is an error"}
// You will find the code in ProcessPayloadWithData function
// 实现了这个接口的错误，panic的时候，就会返回4，同时提示ShowError的内容作为错误消息
type IUserError interface {
	// ShowError Implement this function and return a string as the `message` field of the response message
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
