//go:build linux

package checks

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBrowserProfileTTLMatchesUserAgent: the TTL a profile's connections carry
// is the one the OS in its User-Agent starts with (plus the runner's extra
// hops), checked on the socket the web check really fetched the page over.
func TestBrowserProfileTTLMatchesUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) }))
	defer srv.Close()
	for _, p := range browserProfiles {
		want := 64
		if strings.Contains(p.userAgent(), "Windows NT") {
			want = 128
		}
		assert.Equal(t, want+runnerExtraHops, p.os.ttl(), "%s: User-Agent %q", p.name, p.userAgent())

		tr := newBrowserTransport(p)
		req, err := http.NewRequest(http.MethodGet, srv.URL+"/", nil)
		require.NoError(t, err)
		resp, err := tr.RoundTrip(req)
		require.NoError(t, err)
		_, err = io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		pc := tr.takeIdleH1("http://" + strings.TrimPrefix(srv.URL, "http://"))
		require.NotNil(t, pc, "%s: keep-alive connection", p.name)
		assert.Equal(t, want+runnerExtraHops, sockoptInt(t, pc.conn, syscall.IPPROTO_IP, syscall.IP_TTL), "%s: TTL on the socket", p.name)
		_ = pc.conn.Close()
		tr.CloseIdleConnections()
	}
}
