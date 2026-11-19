package checks

import (
	"net"
	"os"
	"strconv"
	"time"
)

// clientOS is the operating system a check claims to run on, through its
// User-Agent, workstation name or client banner. Its sockets carry that OS's
// initial IP TTL, so the TTL a team sees agrees with the claim, and a TCP
// window clamp that gives the OS's SYN window scale where the host allows.
type clientOS int

const (
	clientLinux clientOS = iota
	clientWindows
	clientMacOS
)

func (o clientOS) String() string {
	switch o {
	case clientWindows:
		return "windows"
	case clientMacOS:
		return "macos"
	}
	return "linux"
}

// runnerExtraHops is how many more routers the runner's traffic crosses to a
// team than a real host at the same source addresses would (environment
// QUOTIENT_RUNNER_EXTRA_HOPS, 0-64, default 0). In Invitational #2 scoring
// SYNs reached teams with TTL 61 while other sources in the same SNAT range
// arrived with 62: one extra hop (the runner pod's node). Set it to 1 there.
var runnerExtraHops = parseExtraHops(os.Getenv("QUOTIENT_RUNNER_EXTRA_HOPS"))

func parseExtraHops(v string) int {
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 || n > 64 {
		return 0
	}
	return n
}

// initialTTL is the TTL the OS starts its packets with.
func (o clientOS) initialTTL() int {
	if o == clientWindows {
		return 128
	}
	return 64
}

// ttl is the TTL the check's packets leave the runner with: the OS's initial
// TTL plus the extra hops, so a team sees what a real host would send.
func (o clientOS) ttl() int {
	return min(o.initialTTL()+runnerExtraHops, 255)
}

// windowClamp caps the TCP receive window so that Linux announces the claimed
// OS's window scale in the SYN: 8 for Windows, 7 for Linux, 6 for macOS.
// Linux derives the scale from the larger of net.ipv4.tcp_rmem[2] and
// net.core.rmem_max, capped by the clamp, so a clamp can only lower it: where
// both sysctls are below 8 MiB (scale 7, the Linux default of 6 MiB) a
// Windows socket still announces 7.
func (o clientOS) windowClamp() int {
	switch o {
	case clientWindows:
		return 8 << 20
	case clientMacOS:
		return 2 << 20
	}
	return 4 << 20
}

// dialer returns a net.Dialer whose sockets carry o's TTL and window clamp.
func (o clientOS) dialer(timeout time.Duration) *net.Dialer {
	return &net.Dialer{Timeout: timeout, Control: o.control}
}
