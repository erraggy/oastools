package mcpserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// isBlockedIP returns true if the IP is private, loopback, link-local, or unspecified.
func isBlockedIP(ip net.IP) bool {
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()
}

// newSafeHTTPClient creates an HTTP client that blocks requests to
// private/loopback/link-local IPs. Used by the MCP server to prevent
// SSRF when resolving specs from URLs provided by AI agents.
func newSafeHTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
				if err != nil {
					return nil, err
				}
				for _, ipAddr := range ips {
					if isBlockedIP(ipAddr.IP) {
						return nil, fmt.Errorf("blocked request to private/loopback IP: %s (%s)", host, ipAddr.IP)
					}
				}
				if len(ips) == 0 {
					return nil, fmt.Errorf("no IP addresses found for host: %s", host)
				}
				// Dial the first resolved address.
				return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			// Re-resolve and check the redirect target.
			host := req.URL.Hostname()
			ips, err := net.DefaultResolver.LookupIPAddr(req.Context(), host)
			if err != nil {
				return err
			}
			for _, ipAddr := range ips {
				if isBlockedIP(ipAddr.IP) {
					return fmt.Errorf("redirect to private/loopback IP blocked: %s (%s)", host, ipAddr.IP)
				}
			}
			return nil
		},
	}
}
