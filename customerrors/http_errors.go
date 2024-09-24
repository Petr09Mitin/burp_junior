package customerrors

import (
	"encoding/json"
)

var HTTPErrors = map[error]int{}

func ParseHTTPError(err error) (msg string, status int) {
	if err == nil {
		err = ErrInternal
	}

	if err.Error() == "" {
		err = ErrInternal
	}

	status, ok := HTTPErrors[err]
	if !ok {
		status = 500
		err = ErrInternal
	}

	msg = err.Error()

	return
}

func MarshalError(err error) (data []byte, marshalErr error) {
	data, marshalErr = json.Marshal(map[string]string{"error": err.Error()})
	return
}
