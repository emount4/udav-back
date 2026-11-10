package lifecycle

type Resource interface {
	Close() error
}
