package params

import "context"

type contextKey struct{}

var paramsKey = &contextKey{}

// WithParams stores URL path parameters in the context.
func WithParams(ctx context.Context, p map[string]string) context.Context {
	return context.WithValue(ctx, paramsKey, p)
}

// GetParams retrieves all URL path parameters from the context.
// Returns nil if no parameters are stored.
func GetParams(ctx context.Context) map[string]string {
	p, _ := ctx.Value(paramsKey).(map[string]string)
	return p
}

// GetParam retrieves a single URL path parameter by name.
// Returns empty string if the parameter is not found.
func GetParam(ctx context.Context, name string) string {
	p := GetParams(ctx)
	if p == nil {
		return ""
	}
	return p[name]
}
