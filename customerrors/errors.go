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
)

var (
	ErrInternal        = NewCustomError(errors.New(ErrInternalMessage))
	ErrJSONMarshalling = NewCustomError(errors.New(ErrJSONMarshallingMessage))
)
