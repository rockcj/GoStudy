package main

import (
	"fmt"
	"regexp"
	"strings"
)

// serverLine matches "server: host" only inside an inline proxy map
// (", server:" or "{ server:"), never "proxy-server-nameserver:".
var serverLine = regexp.MustCompile(`([,{]\s*)server:\s*[^,}\n]+`)

// patchProfile rewrites the handful of top-level keys that must differ from
// the vendor's Windows profile, optionally replaces the (DNS-poisoned) node
// domains with a reachable IP, and enables TUN.
func patchProfile(raw string, port int, controller, secret, serverHost string, tun bool) string {
	if serverHost != "" {
		raw = serverLine.ReplaceAllString(raw, "${1}server: "+serverHost)
	}

	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines)+16)
	hasSecret := false
	hasDNSListen := strings.Contains(raw, "listen: 127.0.0.1:53")

	for _, ln := range lines {
		trimmed := strings.TrimSpace(ln)
		switch {
		case strings.HasPrefix(trimmed, "mixed-port:"):
			ln = fmt.Sprintf("mixed-port: %d", port)
		case strings.HasPrefix(trimmed, "allow-lan:"):
			ln = "allow-lan: false"
		case strings.HasPrefix(trimmed, "external-controller:"):
			ln = fmt.Sprintf("external-controller: '127.0.0.1:%s'", controller)
		case strings.HasPrefix(trimmed, "secret:"):
			ln = "secret: '" + secret + "'"
			hasSecret = true
		case strings.HasPrefix(trimmed, "log-level:"):
			ln = "log-level: warning"
		case strings.HasPrefix(trimmed, "mode:"):
			ln = "mode: rule"
		case strings.HasPrefix(trimmed, "respect-rules:"):
			ln = "    respect-rules: false"
		case strings.HasPrefix(trimmed, "nameserver:"):
			ln = "    nameserver: ['https://doh.pub/dns-query', 'https://dns.alidns.com/dns-query']"
		case strings.HasPrefix(trimmed, "fallback:"):
			ln = "    fallback: ['https://dns.google/dns-query', 'https://cloudflare-dns.com/dns-query']"
		case tun && trimmed == "dns:" && !hasDNSListen:
			out = append(out, ln, "    listen: 127.0.0.1:53")
			hasDNSListen = true
			continue
		}
		out = append(out, ln)
	}

	if !hasSecret {
		for i, ln := range out {
			if strings.HasPrefix(strings.TrimSpace(ln), "external-controller:") {
				out = append(out[:i+1], append([]string{"secret: '" + secret + "'"}, out[i+1:]...)...)
				break
			}
		}
	}

	if tun {
		out = append(out,
			"",
			"tun:",
			"  enable: true",
			"  stack: mixed",
			"  device: freecat",
			"  auto-route: true",
			"  auto-detect-interface: true",
			"  dns-hijack:",
			"    - any:53",
		)
	}

	return strings.Join(out, "\n") + "\n"
}
