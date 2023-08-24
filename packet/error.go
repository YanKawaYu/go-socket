package packet

import "errors"

var (
	badMsgTypeError        = "message type is invalid"
	badLengthEncodingError = "remaining length field exceeded maximum of 4 bytes"
	badReturnCodeError     = "return code is invalid"
	dataExceedsPacketError = "data exceeds packet length"
	msgTooLongError        = "message is too long"
	invalidFlagError       = "flag is invalid"
	invalidProNameError    = "protocol name is invalid"
	invalidProVersionError = "protocol version is invalid"
)

// MessageErr wraps an error that caused a problem that needs to bail out of the
// API, such that errors can be recovered and returned as errors from the
// public API.
type MessageErr struct {
	err error
}

func NewMessageError(err string) MessageErr {
	return MessageErr{
		err: errors.New(err),
	}
}

func (p MessageErr) Error() string {
	return p.err.Error()
}

func GetRecoverError(r interface{}) error {
	var err error
	switch x := r.(type) {
	case string:
		err = errors.New(x)
	case MessageErr:
		err = x
	case error:
		err = x
	default:
		err = errors.New("unknown panic")
	}
	return err
}
