package auth

import "context"

type contextKey string

const (
	UserIDContextKey   contextKey = "user_id"
	UsernameContextKey contextKey = "username"
	EmailContextKey    contextKey = "email"
)

func GetUserID(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(UserIDContextKey).(string)
	return val, ok
}

func GetUsername(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(UsernameContextKey).(string)
	return val, ok
}

func GetEmail(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(EmailContextKey).(string)
	return val, ok
}
