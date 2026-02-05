package ports

// DenialHandler is called when a policy check denies a request.
// Implementations can log, collect metrics, or take other actions.
type DenialHandler interface {
	// OnDenial is called when a capability request is denied.
	// kind: "network", "fs", "env", "exec", "kv"
	// request: the denied request (type depends on kind)
	// reason: human-readable denial reason
	OnDenial(kind string, request interface{}, reason string)
}
