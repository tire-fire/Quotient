//go:build linux

package checks

import (
	"strings"
	"syscall"
)

// control sets o's TTL/hop limit, its DSCP mark (profileMark) and, on TCP, its
// window clamp and SO_RCVBUF before connect, so the SYN carries them.
func (o clientOS) control(network, _ string, c syscall.RawConn) error {
	var serr error
	err := c.Control(func(fd uintptr) {
		s := int(fd)
		v6 := strings.HasSuffix(network, "6")
		if v6 {
			serr = syscall.SetsockoptInt(s, syscall.IPPROTO_IPV6, syscall.IPV6_UNICAST_HOPS, o.ttl())
		} else {
			serr = syscall.SetsockoptInt(s, syscall.IPPROTO_IP, syscall.IP_TTL, o.ttl())
		}
		if serr == nil && profileMark {
			if v6 {
				serr = syscall.SetsockoptInt(s, syscall.IPPROTO_IPV6, syscall.IPV6_TCLASS, o.dscp()<<2)
			} else {
				serr = syscall.SetsockoptInt(s, syscall.IPPROTO_IP, syscall.IP_TOS, o.dscp()<<2)
			}
		}
		if serr == nil && strings.HasPrefix(network, "tcp") {
			serr = syscall.SetsockoptInt(s, syscall.IPPROTO_TCP, syscall.TCP_WINDOW_CLAMP, o.windowClamp())
			if serr == nil && o.rcvBuf() > 0 {
				serr = syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_RCVBUF, o.rcvBuf())
			}
		}
	})
	if err != nil {
		return err
	}
	return serr
}
