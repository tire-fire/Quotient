package checks

import (
	"net"
	"os"
	"strconv"
	"strings"
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

// profileMark turns on the DSCP mark that tells a shaping Divisor which OS a
// socket claims (environment QUOTIENT_PROFILE_MARK=dscp). Divisor then sets
// the TTL and rewrites the SYN to that OS's, and clears the mark before
// packets reach a team. Leave it unset without a shaping Divisor (SYN_SHAPE=on),
// or the DSCP values reach the teams.
var profileMark = os.Getenv("QUOTIENT_PROFILE_MARK") == "dscp"

// dscp is the profile mark Divisor's shaping table matches (RFC 2474 pool 2,
// experimental/local use): 7 Windows, 11 Linux, 15 macOS.
func (o clientOS) dscp() int {
	switch o {
	case clientWindows:
		return 7
	case clientMacOS:
		return 15
	}
	return 11
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

// macOSRcvBuf is the SO_RCVBUF a macOS socket asks for. Linux writes its SYN
// window as the socket's receive space rounded down to a multiple of the MSS
// (64240 with MSS 1460 and default buffers); macOS sends 65535. A receive
// buffer of at least 2 MiB makes the space larger than 65535, so the SYN
// carries 65535, and keeps the window scale at 6 (with a locked buffer Linux
// takes the scale from the buffer, capped by the 2 MiB clamp).
const macOSRcvBuf = 2 << 20

// rcvBufFits reports whether net.core.rmem_max (host-wide; not settable per
// network namespace) lets SO_RCVBUF reach macOSRcvBuf. Below it a locked
// smaller buffer would announce a scale under 6, so the socket keeps
// autotuning and its SYN window stays Linux's (a shaping Divisor then writes
// 65535 into the SYN).
var rcvBufFits = func() bool {
	b, err := os.ReadFile("/proc/sys/net/core/rmem_max")
	if err != nil {
		return false
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(b)))
	return err == nil && n >= macOSRcvBuf
}()

// rcvBuf is the SO_RCVBUF o's sockets set before connect; 0: leave the
// kernel's autotuned default (Windows and Linux send Linux's 64240 window,
// which is also Windows 10/11's).
func (o clientOS) rcvBuf() int {
	if o == clientMacOS && rcvBufFits {
		return macOSRcvBuf
	}
	return 0
}

// dialer returns a net.Dialer whose sockets carry o's TTL and window clamp.
func (o clientOS) dialer(timeout time.Duration) *net.Dialer {
	return &net.Dialer{Timeout: timeout, Control: o.control}
}
