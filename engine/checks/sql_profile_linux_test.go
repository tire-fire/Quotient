package checks

import (
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSqlDSNUsesProfileNetwork(t *testing.T) {
	assert.Equal(t, "u:p@quotient-linux(10.100.21.103:3306)/scoring", sqlDSN("u", "p", "10.100.21.103", 3306, "scoring"))
	assert.Equal(t, "u:p@quotient-linux([fd00::5]:3306)/db", sqlDSN("u", "p", "fd00::5", 3306, "db"))
}

// The SQL check's connections carry the Linux profile's TTL and, behind a
// shaping Divisor, the Linux DSCP mark. (The window clamp shows only in the
// SYN: the kernel rewrites it after the handshake; see TestSqlCheckSYNOnTheWire.)
func TestSqlDialCarriesLinuxProfile(t *testing.T) {
	defer func(m bool) { profileMark = m }(profileMark)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	for _, mark := range []bool{false, true} {
		profileMark = mark
		conn, err := sqlDial(t.Context(), ln.Addr().String())
		require.NoError(t, err)
		assert.Equal(t, clientLinux.ttl(), sockoptInt(t, conn, syscall.IPPROTO_IP, syscall.IP_TTL))
		want := 0
		if mark {
			want = clientLinux.dscp() << 2
		}
		assert.Equal(t, want, sockoptInt(t, conn, syscall.IPPROTO_IP, syscall.IP_TOS), "mark %v", mark)
		_ = conn.Close()
	}
}

// TestSqlCheckSYNOnTheWire runs the real SQL check against a listener on lo and
// captures its SYN (needs CAP_NET_RAW): Linux's TTL and window scale 7, not the
// larger scale an unclamped socket takes from the host's or the runner's
// tcp_rmem (8 with the shaping setup's 8 MiB).
func TestSqlCheckSYNOnTheWire(t *testing.T) {
	lo, err := net.InterfaceByName("lo")
	require.NoError(t, err)
	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(htons(syscall.ETH_P_ALL)))
	if err != nil {
		t.Skipf("no packet socket (%v): run as root to check the SYN on the wire", err)
	}
	defer syscall.Close(fd)
	require.NoError(t, syscall.Bind(fd, &syscall.SockaddrLinklayer{Protocol: htons(syscall.ETH_P_ALL), Ifindex: lo.Index}))
	tv := syscall.NsecToTimeval(int64(200 * time.Millisecond))
	require.NoError(t, syscall.SetsockoptTimeval(fd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &tv))

	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	sports := make(chan uint16, 4)
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			sports <- uint16(c.RemoteAddr().(*net.TCPAddr).Port)
			_ = c.Close()
		}
	}()
	port := ln.Addr().(*net.TCPAddr).Port

	c := &Sql{Service: Service{CredLists: []string{"l"}}, Kind: "mysql", Query: []queryData{{Command: "SELECT 1", Database: "d", Output: "1"}}}
	require.NoError(t, c.Verify("box01", "127.0.0.1", 1, 5, 0, 0))
	c.Port = port
	c.TaskCredentials = []TaskCredential{{Username: "u", Password: "p", Credlist: "l"}}
	results := make(chan Result, 1)
	c.Run(1, "01", 1, results)
	assert.False(t, (<-results).Status, "the listener speaks no MySQL")

	sport := <-sports
	ttl, scale, _ := captureSYN(t, fd, sport, uint16(port))
	assert.Equal(t, clientLinux.ttl(), ttl, "TTL in the SQL check's SYN")
	assert.Equal(t, min(7, hostWindowScale(t)), scale, "window scale in the SQL check's SYN")
}
