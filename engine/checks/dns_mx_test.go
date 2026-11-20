package checks

import (
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startDnsServer serves rr (with the question's name prepended) as the sole
// answer to every query of any type and returns the server's UDP port.
func startDnsServer(t *testing.T, rr string) int {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := &dns.Server{PacketConn: pc, Handler: dns.HandlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(req)
		if len(req.Question) == 1 {
			answer, _ := dns.NewRR(req.Question[0].Name + " 60 IN " + rr)
			m.Answer = append(m.Answer, answer)
		}
		_ = w.WriteMsg(m)
	})}
	go func() { _ = srv.ActivateAndServe() }()
	t.Cleanup(func() { _ = srv.Shutdown() })
	return pc.LocalAddr().(*net.UDPAddr).Port
}

func runDnsRecord(t *testing.T, port int, record DnsRecord) Result {
	t.Helper()
	c := &Dns{Record: []DnsRecord{record}}
	require.NoError(t, c.Verify("box01", "127.0.0.1", 10, 5, 0, 0))
	c.Port = port

	results := make(chan Result, 1)
	c.Run(1, "01", 1, results)
	select {
	case r := <-results:
		return r
	case <-time.After(10 * time.Second):
		t.Fatal("no result")
		return Result{}
	}
}

func TestDnsMxAnswerMatchesExpectedExchanger(t *testing.T) {
	port := startDnsServer(t, "MX 10 Mail.Team01.example.com.")

	for _, tc := range []struct {
		expected string
		pass     bool
	}{
		{"mail.team_.example.com", true},
		{"mail.team_.example.com.", true},
		{"other.team_.example.com", false},
	} {
		r := runDnsRecord(t, port, DnsRecord{Kind: "MX", Domain: "team_.example.com", Answer: []string{tc.expected}})
		assert.Equal(t, tc.pass, r.Status, "expected %q: error=%q debug=%q", tc.expected, r.Error, r.Debug)
	}
}

// A team server must answer an A query with an A record; an MX answer whose
// exchanger spells the expected address is not a correct A answer.
func TestDnsARecordRejectsMxAnswer(t *testing.T) {
	port := startDnsServer(t, "MX 10 192.0.2.10.")

	r := runDnsRecord(t, port, DnsRecord{Kind: "A", Domain: "www.team_.example.com", Answer: []string{"192.0.2.10"}})
	assert.False(t, r.Status, "error=%q debug=%q", r.Error, r.Debug)
}

func TestDnsVerifyRejectsUnsupportedKind(t *testing.T) {
	c := &Dns{Record: []DnsRecord{{Kind: "AAAA", Domain: "team_.example.com", Answer: []string{"::1"}}}}
	err := c.Verify("box01", "127.0.0.1", 10, 5, 0, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported record kind")
}
