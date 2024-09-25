package customerrors

import "errors"

type HTTPError struct {
	Error string `json:"error"`
}

type CustomError struct {
	error
}

func NewCustomError(err error) CustomError {
	if err == nil {
		return CustomError{errors.New("")}
	}

	return CustomError{err}
}

var (
	ErrInternalMessage        = "internal error"
	ErrJSONMarshallingMessage = "error marshalling json"
	ErrParsingFormDataMessage = "error parsing form data"
	ErrParsingRequestMessage  = "error parsing request"
	ErrServingConnectMessage  = "error serving connect"
	ErrSendingRequestMessage  = "error sending request"
	ErrParsingResponseMessage = "error parsing response"
	ErrServingResponseMessage = "error serving response"
	ErrSavingResponseMessage  = "error saving response"
)

var (
	ErrInternal        = NewCustomError(errors.New(ErrInternalMessage))
	ErrJSONMarshalling = NewCustomError(errors.New(ErrJSONMarshallingMessage))
	ErrParsingFormData = NewCustomError(errors.New(ErrParsingFormDataMessage))
	ErrParsingRequest  = NewCustomError(errors.New(ErrParsingRequestMessage))
	ErrServingConnect  = NewCustomError(errors.New(ErrServingConnectMessage))
	ErrSendingRequest  = NewCustomError(errors.New(ErrSendingRequestMessage))
	ErrParsingResponse = NewCustomError(errors.New(ErrParsingResponseMessage))
	ErrServingResponse = NewCustomError(errors.New(ErrServingResponseMessage))
	ErrSavingResponse  = NewCustomError(errors.New(ErrSavingResponseMessage))
)
