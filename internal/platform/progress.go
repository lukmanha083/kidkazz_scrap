package platform

import "context"

// ProgressFunc is a callback for reporting progress messages.
type ProgressFunc func(msg string)

type progressKey struct{}

// WithProgress returns a context carrying the given progress callback.
func WithProgress(ctx context.Context, fn ProgressFunc) context.Context {
	return context.WithValue(ctx, progressKey{}, fn)
}

// ReportProgress calls the progress callback in ctx, if any.
// Safe to call when no callback is set (e.g. MCP mode) — it simply returns.
func ReportProgress(ctx context.Context, msg string) {
	if fn, ok := ctx.Value(progressKey{}).(ProgressFunc); ok && fn != nil {
		fn(msg)
	}
}
