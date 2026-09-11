package validator

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"syscall"
	"time"
)

// URLOptions configures URL validation behavior
type URLOptions struct {
	// AllowedSchemes specifies allowed URL schemes (default: ["http", "https"])
	AllowedSchemes []string
	// AllowLocalhost allows localhost and 127.0.0.1 (default: false)
	AllowLocalhost bool
	// AllowPrivateIP allows private IP addresses (default: false)
	AllowPrivateIP bool
	// ResolveHostTimeout is the timeout for resolving a hostname to check its
	// addresses. Zero means "use the default" (5s), not "skip resolving":
	// leaving a field at its zero value must not switch a security check off.
	// Use DisableHostResolution to opt out deliberately.
	ResolveHostTimeout time.Duration

	// DisableHostResolution skips DNS resolution, so a hostname is accepted on
	// its scheme alone. Only set this when something else performs the check --
	// SSRFDialControl, for instance.
	DisableHostResolution bool
}

// defaultURLOptions returns default URL validation options
func defaultURLOptions() *URLOptions {
	return &URLOptions{
		AllowedSchemes:     []string{"http", "https"},
		AllowLocalhost:     false,
		AllowPrivateIP:     false,
		ResolveHostTimeout: 5 * time.Second,
	}
}

// normalizeURLOptions merges caller-provided options with secure defaults.
//
// Every field here has to distinguish "not set" from "set to the zero value",
// or a caller who overrides one field silently gets the zero value for the
// rest. ResolveHostTimeout was the dangerous case: it was copied
// unconditionally, and zero meant "skip DNS resolution", so
//
//	&URLOptions{AllowedSchemes: []string{"https"}}
//
// turned the hostname SSRF check off entirely while looking like it only
// narrowed the scheme list. Zero now means "use the default"; opting out is
// spelled DisableHostResolution.
func normalizeURLOptions(opts *URLOptions) *URLOptions {
	normalized := defaultURLOptions()
	if opts == nil {
		return normalized
	}

	if opts.AllowedSchemes != nil {
		normalized.AllowedSchemes = opts.AllowedSchemes
	}
	normalized.AllowLocalhost = opts.AllowLocalhost
	normalized.AllowPrivateIP = opts.AllowPrivateIP
	normalized.DisableHostResolution = opts.DisableHostResolution
	if opts.ResolveHostTimeout > 0 {
		normalized.ResolveHostTimeout = opts.ResolveHostTimeout
	}

	return normalized
}

// ValidateURL validates a URL string with SSRF protection
//
// This function performs strict validation on URLs, including:
// - Scheme validation (default: only http and https)
// - Host validation
// - Private IP address blocking (default: blocked)
// - Localhost blocking (default: blocked)
//
// Parameters:
//   - urlStr: URL string to validate
//   - opts: Optional validation options (nil uses secure defaults)
//
// Returns:
//   - error: Returns error if URL is invalid or has security risks; otherwise returns nil
func ValidateURL(urlStr string, opts *URLOptions) error {
	if urlStr == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	opts = normalizeURLOptions(opts)

	// Parse URL
	u, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}
	if u.User != nil {
		return fmt.Errorf("URL cannot include user info")
	}

	// Validate scheme
	if len(opts.AllowedSchemes) > 0 {
		schemeAllowed := false
		for _, allowedScheme := range opts.AllowedSchemes {
			if strings.EqualFold(u.Scheme, allowedScheme) {
				schemeAllowed = true
				break
			}
		}
		if !schemeAllowed {
			return fmt.Errorf("scheme %q is not allowed, allowed schemes: %v", u.Scheme, opts.AllowedSchemes)
		}
	}

	// Validate host
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("URL must contain a valid host")
	}

	// Check localhost. The trailing dot and the ".localhost" suffix both
	// resolve to the loopback interface, so a bare string compare against
	// "localhost" was not enough on its own.
	if !opts.AllowLocalhost {
		hostLower := strings.ToLower(strings.TrimSuffix(host, "."))
		if hostLower == "localhost" || strings.HasSuffix(hostLower, ".localhost") ||
			hostLower == "127.0.0.1" || hostLower == "::1" {
			return fmt.Errorf("access to localhost is not allowed")
		}
	}

	// Parse IP address or resolve hostname for SSRF protection
	ip := net.ParseIP(host)
	if ip != nil {
		if err := checkIPAllowed(ip, opts); err != nil {
			return err
		}
		return nil
	}

	// Host is a hostname: resolve and check every address it maps to.
	//
	// NOTE: this is a check-then-use. The caller resolves the name again when
	// it actually connects, and a short-TTL record can answer differently the
	// second time (DNS rebinding): public address here, 169.254.169.254 there.
	// Use SSRFDialControl on the dialer that performs the request to close
	// that window; ValidateURL alone cannot.
	if !opts.DisableHostResolution && opts.ResolveHostTimeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), opts.ResolveHostTimeout)
		defer cancel()
		resolver := &net.Resolver{}
		addrs, err := resolver.LookupIPAddr(ctx, host)
		if err != nil {
			return fmt.Errorf("failed to resolve host %q: %w", host, err)
		}
		if len(addrs) == 0 {
			return fmt.Errorf("host %q resolved to no addresses", host)
		}
		for _, ipAddr := range addrs {
			if err := checkIPAllowed(ipAddr.IP, opts); err != nil {
				return err
			}
		}
	}

	return nil
}

// checkIPAllowed returns an error if the IP is not allowed by opts (loopback/private checks).
func checkIPAllowed(ip net.IP, opts *URLOptions) error {
	if isAlwaysBlockedIP(ip) {
		return fmt.Errorf("access to non-routable IP address is not allowed: %s", ip.String())
	}

	if !opts.AllowLocalhost && ip.IsLoopback() {
		return fmt.Errorf("access to loopback address is not allowed: %s", ip.String())
	}
	if !opts.AllowPrivateIP && isPrivateIP(ip) {
		if opts.AllowLocalhost && ip.IsLoopback() {
			return nil
		}
		return fmt.Errorf("access to private IP address is not allowed: %s", ip.String())
	}
	return nil
}

// isAlwaysBlockedIP checks IP ranges that should always be rejected for outbound URL targets.
func isAlwaysBlockedIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	// Unspecified and multicast are never valid remote service targets.
	if ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}

	// 0.0.0.0/8 and 255.255.255.255 are non-routable special addresses.
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 0 || (ip4[0] == 255 && ip4[1] == 255 && ip4[2] == 255 && ip4[3] == 255) {
			return true
		}
	}

	return false
}

// isPrivateIP checks if IP is an internal/non-public address
//
// Included ranges:
// - 10.0.0.0/8 (10.0.0.0 to 10.255.255.255)
// - 172.16.0.0/12 (172.16.0.0 to 172.31.255.255)
// - 192.168.0.0/16 (192.168.0.0 to 192.168.255.255)
// - 127.0.0.0/8 (127.0.0.0 to 127.255.255.255) - loopback address
// - 100.64.0.0/10 (carrier-grade NAT)
// - 169.254.0.0/16 (link-local, includes cloud metadata endpoints)
// - 198.18.0.0/15 (benchmark testing)
// - IPv6 ULA/link-local/loopback
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	// Built-in checks cover RFC1918 + IPv6 ULA + loopback/link-local ranges.
	if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	if ip4 := ip.To4(); ip4 != nil {
		// RFC6598 shared address space for carrier-grade NAT
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
		// IPv4 link-local range (often abused for metadata SSRF)
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
		// Benchmark testing range
		if ip4[0] == 198 && (ip4[1] == 18 || ip4[1] == 19) {
			return true
		}
		// IETF protocol assignments (192.0.0.0/24) and TEST-NET-1
		// (192.0.2.0/24) are not routable destinations.
		if ip4[0] == 192 && ip4[1] == 0 && (ip4[2] == 0 || ip4[2] == 2) {
			return true
		}
		// Reserved for future use, 240.0.0.0/4.
		if ip4[0] >= 240 {
			return true
		}
	}

	return false
}

// SSRFDialControl returns a function for net.Dialer.Control that rejects a
// connection whose resolved address is not allowed by opts.
//
// This is the half of SSRF protection that ValidateURL cannot provide. The
// validator inspects the addresses a name resolved to at validation time; the
// Control hook runs against the address actually being connected to, after the
// resolution that the request itself performed, so a rebinding answer is
// caught.
//
//	dialer := &net.Dialer{Control: validator.SSRFDialControl(opts)}
//	transport := http.DefaultTransport.(*http.Transport).Clone()
//	transport.DialContext = dialer.DialContext
func SSRFDialControl(opts *URLOptions) func(network, address string, c syscall.RawConn) error {
	normalized := normalizeURLOptions(opts)

	return func(_, address string, _ syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return fmt.Errorf("unable to parse dial address %q: %w", address, err)
		}
		ip := net.ParseIP(host)
		if ip == nil {
			return fmt.Errorf("dial address %q is not an IP", host)
		}
		return checkIPAllowed(ip, normalized)
	}
}
