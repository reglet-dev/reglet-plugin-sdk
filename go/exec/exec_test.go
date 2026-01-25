package exec

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/reglet-dev/reglet-sdk/go/domain/entities"
	"github.com/reglet-dev/reglet-sdk/go/domain/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mock CommandRunner ---

type MockCommandRunner struct {
	mock.Mock
}

func (m *MockCommandRunner) Run(ctx context.Context, req ports.CommandRequest) (*ports.CommandResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.CommandResult), args.Error(1)
}

// --- Run Functional Tests ---

func TestRun_WithMockRunner(t *testing.T) {
	mockRunner := new(MockCommandRunner)

	expectedReq := CommandRequest{
		Command: "echo",
		Args:    []string{"hello"},
		Timeout: 30, // default
	}

	expectedRes := &CommandResponse{
		Stdout:   "hello\n",
		ExitCode: 0,
	}

	mockRunner.On("Run", mock.Anything, expectedReq).Return(expectedRes, nil)

	resp, err := Run(context.Background(), CommandRequest{
		Command: "echo",
		Args:    []string{"hello"},
	}, WithRunner(mockRunner))

	require.NoError(t, err)
	assert.Equal(t, "hello\n", resp.Stdout)
	mockRunner.AssertExpectations(t)
}

func TestRun_WithWorkdir(t *testing.T) {
	mockRunner := new(MockCommandRunner)

	expectedReq := CommandRequest{
		Command: "ls",
		Dir:     "/tmp",
		Timeout: 30,
	}

	mockRunner.On("Run", mock.Anything, expectedReq).Return(&CommandResponse{}, nil)

	_, err := Run(context.Background(), CommandRequest{Command: "ls"},
		WithWorkdir("/tmp"),
		WithRunner(mockRunner),
	)

	require.NoError(t, err)
	mockRunner.AssertExpectations(t)
}

func TestRun_WithEnv(t *testing.T) {
	mockRunner := new(MockCommandRunner)

	expectedReq := CommandRequest{
		Command: "env",
		Env:     []string{"FOO=bar"},
		Timeout: 30,
	}

	mockRunner.On("Run", mock.Anything, expectedReq).Return(&CommandResponse{}, nil)

	_, err := Run(context.Background(), CommandRequest{Command: "env"},
		WithEnv([]string{"FOO=bar"}),
		WithRunner(mockRunner),
	)

	require.NoError(t, err)
	mockRunner.AssertExpectations(t)
}

func TestRun_DefaultRunner_PanicsOnNative(t *testing.T) {
	// This ensures that if we don't inject a mock, we get the stub (on native) which panics
	assert.PanicsWithValue(t, "WASM Exec adapter not available in native build. Use WithCommandRunner() to inject a mock.", func() {
		_, _ = Run(context.Background(), CommandRequest{Command: "ls"})
	})
}

// --- RunOption Functional Options Tests (T014) ---

func TestRunOption_DefaultConfig(t *testing.T) {
	cfg := defaultRunConfig()

	assert.Empty(t, cfg.workdir, "default workdir should be empty (inherit)")
	assert.Nil(t, cfg.env, "default env should be nil (inherit)")
	assert.Equal(t, 30*time.Second, cfg.timeout, "default timeout should be 30s")
}

func TestApplyRunOptions_WithDefaults(t *testing.T) {
	cfg := applyRunOptions()

	assert.Empty(t, cfg.workdir)
	assert.Nil(t, cfg.env)
	assert.Equal(t, 30*time.Second, cfg.timeout)
}

func TestApplyRunOptions_WithWorkdir(t *testing.T) {
	cfg := applyRunOptions(WithWorkdir("/tmp/work"))

	assert.Equal(t, "/tmp/work", cfg.workdir)
	assert.Nil(t, cfg.env)
	assert.Equal(t, 30*time.Second, cfg.timeout)
}

func TestApplyRunOptions_WithEnv(t *testing.T) {
	env := []string{"FOO=bar", "BAZ=qux"}
	cfg := applyRunOptions(WithEnv(env))

	assert.Equal(t, env, cfg.env)
}

func TestApplyRunOptions_WithExecTimeout(t *testing.T) {
	cfg := applyRunOptions(WithExecTimeout(60 * time.Second))

	assert.Equal(t, 60*time.Second, cfg.timeout)
}

func TestApplyRunOptions_WithExecTimeout_IgnoresInvalid(t *testing.T) {
	tests := []struct {
		name     string
		timeout  time.Duration
		expected time.Duration
	}{
		{"zero should use default", 0, 30 * time.Second},
		{"negative should use default", -1 * time.Second, 30 * time.Second},
		{"positive should be applied", 45 * time.Second, 45 * time.Second},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := applyRunOptions(WithExecTimeout(tc.timeout))
			assert.Equal(t, tc.expected, cfg.timeout)
		})
	}
}

func TestApplyRunOptions_MultipleOptions(t *testing.T) {
	env := []string{"PATH=/usr/bin"}
	cfg := applyRunOptions(
		WithWorkdir("/home/user"),
		WithEnv(env),
		WithExecTimeout(15*time.Second),
	)

	assert.Equal(t, "/home/user", cfg.workdir)
	assert.Equal(t, env, cfg.env)
	assert.Equal(t, 15*time.Second, cfg.timeout)
}

func TestApplyRunOptions_OptionsApplyInOrder(t *testing.T) {
	cfg := applyRunOptions(
		WithWorkdir("/first"),
		WithWorkdir("/second"),
	)

	assert.Equal(t, "/second", cfg.workdir, "last option should win")
}

// --- Original Wire Format Tests ---

// Note: Actual command execution requires WASM runtime with host functions.
// These tests focus on wire format structures and data serialization.

func TestCommandRequest_Serialization(t *testing.T) {
	tests := []struct {
		name    string
		request entities.WireExecRequest
	}{
		{
			name: "simple command",
			request: entities.WireExecRequest{
				Command: "/usr/bin/whoami",
			},
		},
		{
			name: "command with args",
			request: entities.WireExecRequest{
				Command: "/bin/echo",
				Args:    []string{"hello", "world"},
			},
		},
		{
			name: "command with environment",
			request: entities.WireExecRequest{
				Command: "/usr/bin/env",
				Env:     []string{"FOO=bar", "BAZ=qux"},
			},
		},
		{
			name: "command with working directory",
			request: entities.WireExecRequest{
				Command: "/usr/bin/ls",
				Args:    []string{" -la"},
				Dir:     "/tmp",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.request)
			require.NoError(t, err)

			var decoded entities.WireExecRequest
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			assert.Equal(t, tt.request.Command, decoded.Command)
			assert.Equal(t, tt.request.Args, decoded.Args)
			assert.Equal(t, tt.request.Env, decoded.Env)
			assert.Equal(t, tt.request.Dir, decoded.Dir)
		})
	}
}

func TestCommandResponse_Serialization(t *testing.T) {
	tests := []struct {
		name     string
		response entities.ExecResponse
	}{
		{
			name: "successful execution",
			response: entities.ExecResponse{
				Stdout:     "command output",
				Stderr:     "",
				ExitCode:   0,
				DurationMs: 123,
			},
		},
		{
			name: "failed execution",
			response: entities.ExecResponse{
				Stdout:   "",
				Stderr:   "error: command not found",
				ExitCode: 127,
			},
		},
		{
			name: "timeout execution",
			response: entities.ExecResponse{
				Stdout:     "partial output",
				Stderr:     "",
				ExitCode:   -1,
				DurationMs: 5000,
				IsTimeout:  true,
			},
		},
		{
			name: "execution with mixed output",
			response: entities.ExecResponse{
				Stdout:     "normal output\n",
				Stderr:     "warning message\n",
				ExitCode:   0,
				DurationMs: 456,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.response)
			require.NoError(t, err)

			var decoded entities.ExecResponse
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			assert.Equal(t, tt.response.Stdout, decoded.Stdout)
			assert.Equal(t, tt.response.Stderr, decoded.Stderr)
			assert.Equal(t, tt.response.ExitCode, decoded.ExitCode)
			assert.Equal(t, tt.response.DurationMs, decoded.DurationMs)
			assert.Equal(t, tt.response.IsTimeout, decoded.IsTimeout)
		})
	}
}

func TestCommandExitCodes(t *testing.T) {
	exitCodes := []struct {
		code    int
		meaning string
	}{
		{0, "success"},
		{1, "general error"},
		{2, "misuse of shell command"},
		{126, "command cannot execute"},
		{127, "command not found"},
		{130, "terminated by Ctrl+C"},
		{137, "killed (SIGKILL)"},
		{143, "terminated (SIGTERM)"},
	}

	for _, ec := range exitCodes {
		t.Run(ec.meaning, func(t *testing.T) {
			resp := entities.ExecResponse{
				ExitCode: ec.code,
			}

			data, err := json.Marshal(resp)
			require.NoError(t, err)

			var decoded entities.ExecResponse
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)
			assert.Equal(t, ec.code, decoded.ExitCode)
		})
	}
}

func TestCommandWithWireformatError(t *testing.T) {
	// Test that CommandResponse can include structured errors
	// (though typically errors are in the ExitCode/Stderr)

	// Note: This documents how errors could be structured
	// Current implementation uses ExitCode + Stderr for errors
	tests := []struct {
		name      string
		errorType string
		errorCode string
		message   string
	}{
		{
			name:      "permission denied",
			errorType: "execution",
			errorCode: "EACCES",
			message:   "permission denied",
		},
		{
			name:      "timeout",
			errorType: "timeout",
			errorCode: "ETIMEDOUT",
			message:   "command execution timed out",
		},
		{
			name:      "not found",
			errorType: "execution",
			errorCode: "ENOENT",
			message:   "command not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Documenting that errors would use ErrorDetail structure
			errorDetail := &entities.ErrorDetail{
				Type:    tt.errorType,
				Code:    tt.errorCode,
				Message: tt.message,
			}

			data, err := json.Marshal(errorDetail)
			require.NoError(t, err)

			var decoded entities.ErrorDetail
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			assert.Equal(t, tt.errorType, decoded.Type)
			assert.Equal(t, tt.errorCode, decoded.Code)
			assert.Equal(t, tt.message, decoded.Message)
		})
	}
}

func TestCommandDurationTracking(t *testing.T) {
	// Test that duration is properly tracked in ExecResponseWire
	durations := []int64{0, 10, 100, 1000, 5000, 30000}

	for _, duration := range durations {
		t.Run(string(rune(duration)), func(t *testing.T) {
			resp := entities.ExecResponse{
				ExitCode:   0,
				DurationMs: duration,
			}

			data, err := json.Marshal(resp)
			require.NoError(t, err)

			var decoded entities.ExecResponse
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)
			assert.Equal(t, duration, decoded.DurationMs)
		})
	}
}