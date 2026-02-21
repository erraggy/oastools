package mcpserver

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsBlockedIP(t *testing.T) {
	tests := []struct {
		ip      string
		blocked bool
	}{
		{"127.0.0.1", true},      // loopback
		{"10.0.0.1", true},       // private (Class A)
		{"172.16.0.1", true},     // private (Class B)
		{"192.168.1.1", true},    // private (Class C)
		{"169.254.1.1", true},    // link-local
		{"::1", true},            // IPv6 loopback
		{"0.0.0.0", true},        // unspecified IPv4
		{"::", true},             // unspecified IPv6
		{"fe80::1", true},        // IPv6 link-local
		{"fd00::1", true},        // IPv6 ULA (private)
		{"8.8.8.8", false},       // public (Google DNS)
		{"1.1.1.1", false},       // public (Cloudflare DNS)
		{"93.184.216.34", false}, // public (example.com)
	}
	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			require.NotNil(t, ip, "failed to parse IP: %s", tt.ip)
			assert.Equal(t, tt.blocked, isBlockedIP(ip))
		})
	}
}

func TestNewSafeHTTPClient(t *testing.T) {
	client := newSafeHTTPClient()
	require.NotNil(t, client)
	assert.NotZero(t, client.Timeout)
	assert.NotNil(t, client.CheckRedirect)
	assert.NotNil(t, client.Transport)
}
