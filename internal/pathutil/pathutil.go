package pathutil

import (
	"fmt"
	"net/netip"
	"net/url"
	"path/filepath"
	"strings"
)

var privateIPv4Prefixes = []netip.Prefix{
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("100.64.0.0/10"),
}

var privateIPv6Prefixes = []netip.Prefix{
	netip.MustParsePrefix("fc00::/7"),
}

func IsWithinDir(base string, target string) bool {
	if strings.TrimSpace(base) == "" || strings.TrimSpace(target) == "" {
		return false
	}

	basePath, err := filepath.Abs(base)
	if err != nil {
		return false
	}

	targetPath, err := filepath.Abs(target)
	if err != nil {
		return false
	}

	relativePath, err := filepath.Rel(basePath, targetPath)
	if err != nil {
		return false
	}

	return relativePath == "." || (!strings.HasPrefix(relativePath, "..") && relativePath != "..")
}

func NormalizeServerURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("server URL cannot be empty")
	}

	parsedURL, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid server URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("server URL must use http or https")
	}
	if parsedURL.User != nil {
		return "", fmt.Errorf("server URL must not include user info")
	}
	if parsedURL.Host == "" || parsedURL.Hostname() == "" {
		return "", fmt.Errorf("server URL must include a host")
	}
	if parsedURL.Port() == "" {
		return "", fmt.Errorf("server URL must include an explicit port")
	}
	if parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return "", fmt.Errorf("server URL must not include query parameters or fragments")
	}
	if parsedURL.Path != "" && parsedURL.Path != "/" {
		return "", fmt.Errorf("server URL must not include a path")
	}

	host := strings.ToLower(parsedURL.Hostname())
	if host != "localhost" {
		ip, err := netip.ParseAddr(host)
		if err != nil {
			return "", fmt.Errorf("server URL host must be localhost or a private IP address for the MVP")
		}
		if !isAllowedPrivateIP(ip) {
			return "", fmt.Errorf("server URL must target loopback, RFC1918, or Tailscale CGNAT space")
		}
	}

	parsedURL.Path = ""
	parsedURL.RawPath = ""
	return parsedURL.String(), nil
}

func isAllowedPrivateIP(ip netip.Addr) bool {
	if ip.IsLoopback() {
		return true
	}

	prefixes := privateIPv4Prefixes
	if ip.Is6() {
		prefixes = privateIPv6Prefixes
	}

	for _, prefix := range prefixes {
		if prefix.Contains(ip) {
			return true
		}
	}

	return false
}
