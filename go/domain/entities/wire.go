// Package entities defines core domain types and wire protocol structures.
// These types serve dual purpose: domain entities AND JSON wire format DTOs.
package entities

import "time"

// ContextWire is the JSON wire format for context.Context propagation.
type ContextWire struct {
	Deadline  *time.Time `json:"deadline,omitempty"`
	RequestID string     `json:"request_id,omitempty"`
	TimeoutMs int64      `json:"timeout_ms,omitempty"`
	Canceled  bool       `json:"canceled,omitempty"`
}

// DNSRequest is the JSON wire format for a DNS lookup request.
type DNSRequest struct {
	Hostname   string      `json:"hostname"`
	Type       string      `json:"type"`
	Nameserver string      `json:"nameserver,omitempty"`
	Context    ContextWire `json:"context"`
}

// DNSResponse is the JSON wire format for a DNS lookup response.
type DNSResponse struct {
	Error     *ErrorDetail `json:"error,omitempty"`
	Records   []string     `json:"records,omitempty"`
	MXRecords []MXRecord   `json:"mx_records,omitempty"`
}

// MXRecord represents a single MX record.
type MXRecord struct {
	Host string `json:"host"`
	Pref uint16 `json:"pref"`
}

// HTTPRequest is the JSON wire format for an HTTP request.
type HTTPRequest struct {
	Headers map[string][]string `json:"headers,omitempty"`
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Body    string              `json:"body,omitempty"`
	Context ContextWire         `json:"context"`
}

// HTTPResponse is the JSON wire format for an HTTP response.
type HTTPResponse struct {
	Headers       map[string][]string `json:"headers,omitempty"`
	Error         *ErrorDetail        `json:"error,omitempty"`
	Body          string              `json:"body,omitempty"`
	StatusCode    int                 `json:"status_code"`
	BodyTruncated bool                `json:"body_truncated,omitempty"`
}

// TCPRequest is the JSON wire format for a TCP connection request.
type TCPRequest struct {
	Host      string      `json:"host"`
	Port      string      `json:"port"`
	Context   ContextWire `json:"context"`
	TimeoutMs int         `json:"timeout_ms,omitempty"`
	TLS       bool        `json:"tls"`
}

// TCPResponse is the JSON wire format for a TCP connection response.
type TCPResponse struct {
	TLSCertNotAfter *time.Time   `json:"tls_cert_not_after,omitempty"`
	Error           *ErrorDetail `json:"error,omitempty"`
	TLSVersion      string       `json:"tls_version,omitempty"`
	LocalAddr       string       `json:"local_addr,omitempty"`
	TLSCipherSuite  string       `json:"tls_cipher_suite,omitempty"`
	TLSServerName   string       `json:"tls_server_name,omitempty"`
	TLSCertSubject  string       `json:"tls_cert_subject,omitempty"`
	TLSCertIssuer   string       `json:"tls_cert_issuer,omitempty"`
	RemoteAddr      string       `json:"remote_addr,omitempty"`
	Address         string       `json:"address,omitempty"`
	ResponseTimeMs  int64        `json:"response_time_ms,omitempty"`
	TLS             bool         `json:"tls,omitempty"`
	Connected       bool         `json:"connected"`
}

// SMTPRequest is the JSON wire format for an SMTP connection request.
type SMTPRequest struct {
	Host      string      `json:"host"`
	Port      string      `json:"port"`
	Context   ContextWire `json:"context"`
	TimeoutMs int         `json:"timeout_ms,omitempty"`
	TLS       bool        `json:"tls"`
	StartTLS  bool        `json:"starttls"`
}

// SMTPResponse is the JSON wire format for an SMTP connection response.
type SMTPResponse struct {
	Error          *ErrorDetail `json:"error,omitempty"`
	Address        string       `json:"address,omitempty"`
	Banner         string       `json:"banner,omitempty"`
	TLSVersion     string       `json:"tls_version,omitempty"`
	TLSCipherSuite string       `json:"tls_cipher_suite,omitempty"`
	TLSServerName  string       `json:"tls_server_name,omitempty"`
	ResponseTimeMs int64        `json:"response_time_ms,omitempty"`
	Connected      bool         `json:"connected"`
	TLS            bool         `json:"tls,omitempty"`
}

// ExecRequest is the JSON wire format for an exec request.
type ExecRequest struct {
	Args    []string    `json:"args"`
	Env     []string    `json:"env,omitempty"`
	Command string      `json:"command"`
	Dir     string      `json:"dir,omitempty"`
	Context ContextWire `json:"context"`
}

// ExecResponse is the JSON wire format for an exec response.
type ExecResponse struct {
	Error      *ErrorDetail `json:"error,omitempty"`
	Stdout     string       `json:"stdout"`
	Stderr     string       `json:"stderr"`
	ExitCode   int          `json:"exit_code"`
	DurationMs int64        `json:"duration_ms,omitempty"`
	IsTimeout  bool         `json:"is_timeout,omitempty"`
}
