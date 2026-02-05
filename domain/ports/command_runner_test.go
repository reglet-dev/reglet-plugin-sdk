package ports

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCommandRunner is a mock implementation of CommandRunner for testing.
type MockCommandRunner struct {
	RunFunc func(ctx context.Context, req CommandRequest) (*CommandResult, error)
}

func (m *MockCommandRunner) Run(ctx context.Context, req CommandRequest) (*CommandResult, error) {
	if m.RunFunc != nil {
		return m.RunFunc(ctx, req)
	}
	return &CommandResult{
		Stdout:   "default output",
		ExitCode: 0,
	}, nil
}

// Compile-time interface check
var _ CommandRunner = (*MockCommandRunner)(nil)

func TestMockCommandRunner_ImplementsInterface(t *testing.T) {
	var runner CommandRunner = &MockCommandRunner{}
	require.NotNil(t, runner)
}

func TestMockCommandRunner_Run(t *testing.T) {
	ctx := context.Background()

	t.Run("default behavior", func(t *testing.T) {
		mock := &MockCommandRunner{}
		res, err := mock.Run(ctx, CommandRequest{Command: "echo"})

		require.NoError(t, err)
		assert.Equal(t, "default output", res.Stdout)
		assert.Equal(t, 0, res.ExitCode)
	})

	t.Run("custom behavior", func(t *testing.T) {
		mock := &MockCommandRunner{
			RunFunc: func(ctx context.Context, req CommandRequest) (*CommandResult, error) {
				if req.Command == "fail" {
					return &CommandResult{
						Stderr:   "failed",
						ExitCode: 1,
					}, nil
				}
				return &CommandResult{Stdout: "ok"}, nil
			},
		}

		// Success case
		res, err := mock.Run(ctx, CommandRequest{Command: "ok"})
		require.NoError(t, err)
		assert.Equal(t, "ok", res.Stdout)

		// Failure case
		res, err = mock.Run(ctx, CommandRequest{Command: "fail"})
		require.NoError(t, err)
		assert.Equal(t, 1, res.ExitCode)
		assert.Equal(t, "failed", res.Stderr)
	})

	t.Run("error behavior", func(t *testing.T) {
		expectedErr := errors.New("execution error")
		mock := &MockCommandRunner{
			RunFunc: func(ctx context.Context, req CommandRequest) (*CommandResult, error) {
				return nil, expectedErr
			},
		}

		res, err := mock.Run(ctx, CommandRequest{Command: "error"})
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Equal(t, expectedErr, err)
	})
}
