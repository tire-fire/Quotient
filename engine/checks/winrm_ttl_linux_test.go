//go:build linux

package checks

import (
	"net"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The WinRM check claims Windows' own client (User-Agent "Microsoft WinRM
// Client"), so its sockets must carry Windows' TTL, not the runner's Linux 64.
func TestWinRMDialerTTLMatchesUserAgent(t *testing.T) {
	require.Equal(t, "Microsoft WinRM Client", winrmUserAgent)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	conn, err := winrmDialer().Dial("tcp", ln.Addr().String())
	require.NoError(t, err)
	defer conn.Close()
	assert.Equal(t, 128+runnerExtraHops, sockoptInt(t, conn, syscall.IPPROTO_IP, syscall.IP_TTL))
}
