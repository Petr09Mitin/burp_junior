package customerrors

import (
	"encoding/json"
)

var HTTPErrors = map[error]int{
	ErrInternal:        500,
	ErrJSONMarshalling: 500,
	ErrParsingFormData: 400,
	ErrParsingRequest:  400,
	ErrServingConnect:  500,
	ErrSendingRequest:  500,
	ErrParsingResponse: 500,
	ErrServingResponse: 500,
	ErrSavingResponse:  500,
	ErrInvalidRequest:  400,
}

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
