package requestid

import "context"

type requestIDKey struct{}

func WithContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// FromContext возвращает пустую строку, если в контексте нет ID, например при запуске сервера.
func FromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}
