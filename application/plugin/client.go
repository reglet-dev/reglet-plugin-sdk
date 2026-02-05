package plugin

import "context"

// clientKey is the context key for storing the plugin client.
type clientKey struct{}

// WithClient returns a new context with the client attached.
// Called by the SDK wrapper before invoking typed handlers.
func WithClient(ctx context.Context, client any) context.Context {
	return context.WithValue(ctx, clientKey{}, client)
}

// GetClient extracts a typed client from the context.
// Panics if the client is not present or is the wrong type.
//
// Example:
//
//	func (s *DNSService) ResolveHandler(ctx context.Context, in *ResolveInput) (*ResolveOutput, error) {
//	    resolver := plugin.GetClient[ports.DNSResolver](ctx)
//	    records, err := resolver.LookupHost(ctx, in.Hostname)
//	    // ...
//	}
func GetClient[T any](ctx context.Context) T {
	v := ctx.Value(clientKey{})
	if v == nil {
		panic("plugin: no client in context - ensure handler is called via SDK")
	}
	client, ok := v.(T)
	if !ok {
		panic("plugin: client type mismatch")
	}
	return client
}

// TryGetClient extracts a typed client from the context.
// Returns the zero value and false if not present or wrong type.
func TryGetClient[T any](ctx context.Context) (T, bool) {
	var zero T
	v := ctx.Value(clientKey{})
	if v == nil {
		return zero, false
	}
	client, ok := v.(T)
	return client, ok
}
