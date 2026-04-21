package auth

import (
	"log"
	"net"
	"net/http"
	"strings"
)

// RealIPMiddleware returns middleware that sets r.RemoteAddr to the client IP
// from X-Forwarded-For / X-Real-Ip, but only if the direct peer is a trusted proxy.
// If trustedProxies is empty, the middleware is a no-op (RemoteAddr stays unchanged).
func RealIPMiddleware(trustedProxies []string) func(http.Handler) http.Handler {
	if len(trustedProxies) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	nets := make([]*net.IPNet, 0, len(trustedProxies))
	for _, p := range trustedProxies {
		// If it's a plain IP, convert to /32 or /128
		if !strings.Contains(p, "/") {
			if strings.Contains(p, ":") {
				p += "/128"
			} else {
				p += "/32"
			}
		}
		_, cidr, err := net.ParseCIDR(p)
		if err == nil {
			nets = append(nets, cidr)
		} else {
			log.Printf("WARNING: invalid trusted proxy CIDR %q: %v", p, err)
		}
	}

	isTrusted := func(ip net.IP) bool {
		for _, cidr := range nets {
			if cidr.Contains(ip) {
				return true
			}
		}
		return false
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			peerIP := peerAddr(r.RemoteAddr)
			if peerIP != nil && isTrusted(peerIP) {
				if rip := realIP(r); rip != "" {
					r.RemoteAddr = rip
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func realIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first (leftmost) IP — the original client
		if i := strings.IndexByte(xff, ','); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return strings.TrimSpace(xri)
	}
	return ""
}

func peerAddr(remoteAddr string) net.IP {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return net.ParseIP(remoteAddr)
	}
	return net.ParseIP(host)
}
