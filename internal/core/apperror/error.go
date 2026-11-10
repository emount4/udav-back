package apperror

type Kind uint8

const (
	InvalidInput Kind = iota + 1
	Unauthenticated
	Forbidden
	NotFound
	Conflict
	RateLimited
)

// Error описывает ожидаемую ошибку. Код и сообщение входят в публичный контракт API.
type Error struct {
	kind    Kind
	code    string
	message string
}

func New(kind Kind, code, message string) *Error {
	return &Error{kind: kind, code: code, message: message}
}

func (e *Error) Error() string {
	return e.code + ": " + e.message
}

func (e *Error) Kind() Kind {
	return e.kind
}

func (e *Error) Code() string {
	return e.code
}

func (e *Error) Message() string {
	return e.message
}
