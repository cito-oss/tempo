package tasks

const UnknownPersonErrorType = "UnknownPersonError"

type UnknownPersonError struct {
	name string
}

func (e UnknownPersonError) Error() string {
	return `unknown person named "` + e.name + `"`
}

func NewUnknownPersonError(name string) UnknownPersonError {
	return UnknownPersonError{name: name}
}
