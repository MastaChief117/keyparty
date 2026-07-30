package proxy

import (
	"net"
	"net/url"
	"strings"
)

// IsBlockedURL checks if a URL targets a private/loopback/link-local address
// using proper IP parsing instead of naive string prefix matching.
func IsBlockedURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return true
	}

	host := u.Hostname()
	if host == "" {
		return true
	}

	// Block known metadata endpoints by hostname suffix
	lowerHost := strings.ToLower(host)
	if strings.HasSuffix(lowerHost, ".metadata.google") ||
		strings.HasSuffix(lowerHost, ".metadata.google.internal") ||
		strings.Contains(lowerHost, "metadata.google") {
		return true
	}

	// Try to parse as IP address (handles IPv4, IPv6, IPv6-mapped IPv4)
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return true
		}
		return false
	}

	// For hostnames, block localhost and common internal names
	if lowerHost == "localhost" || lowerHost == "localhost.localdomain" {
		return true
	}

	// Resolve hostname and check resolved IPs
	ips, err := net.LookupIP(host)
	if err != nil {
		return true // Block on DNS resolution failure
	}

	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return true
		}
	}

	return false
}
