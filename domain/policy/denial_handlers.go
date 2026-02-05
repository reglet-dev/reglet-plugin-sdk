package policy

import (
	"fmt"
	"os"

	"github.com/reglet-dev/reglet-sdk/domain/ports"
)

// Ensure implementations satisfy the interface.
var _ ports.DenialHandler = (*StderrDenialHandler)(nil)
var _ ports.DenialHandler = (*NopDenialHandler)(nil)

// StderrDenialHandler logs denials to stderr.
type StderrDenialHandler struct{}

func (h *StderrDenialHandler) OnDenial(kind string, request interface{}, reason string) {
	fmt.Fprintf(os.Stderr, "Permission Denied [%s]: %v (Reason: %s)\n", kind, request, reason)
}

// NopDenialHandler does nothing.
type NopDenialHandler struct{}

func (h *NopDenialHandler) OnDenial(kind string, request interface{}, reason string) {}
