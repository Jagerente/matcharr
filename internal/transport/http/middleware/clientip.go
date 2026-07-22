package middleware

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP resolves the address rate limits key on. Header names the field
// a trusted reverse proxy stamps with the real client address
// (CF-Connecting-IP behind Cloudflare). Empty means the socket address is
// the identity: a header set by anyone else is a free way around every
// limit built on it.
type ClientIP struct {
	Header string
}

func (r ClientIP) FromRequest(req *http.Request) string {
	if r.Header != "" {
		if v := req.Header.Get(r.Header); v != "" {
			// X-Forwarded-For carries the whole proxy chain; the client is first.
			if comma := strings.IndexByte(v, ','); comma >= 0 {
				v = v[:comma]
			}
			return strings.TrimSpace(v)
		}
	}
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}
	return host
}
