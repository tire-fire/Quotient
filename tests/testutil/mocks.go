package testutil

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/miekg/dns"
)

// MockWebServer creates a mock web server for testing
type MockWebServer struct {
	Server *httptest.Server
	URL    string
	Port   string
}

// NewMockWebServer creates a new mock web server
func NewMockWebServer(content string) *MockWebServer {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(content)); err != nil {
			slog.Error("failed to write mock response", "error", err)
		}
	}))

	// Extract port from URL
	parts := strings.Split(server.URL, ":")
	port := parts[len(parts)-1]

	return &MockWebServer{
		Server: server,
		URL:    server.URL,
		Port:   port,
	}
}

// Close shuts down the mock web server
func (m *MockWebServer) Close() {
	m.Server.Close()
}

// MockDNSServer creates a mock DNS server for testing
type MockDNSServer struct {
	Address string
	Port    string
	server  *dns.Server
}

// NewMockDNSServer creates a new mock DNS server
func NewMockDNSServer() (*MockDNSServer, error) {
	// Create UDP listener on random port
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	addr := pc.LocalAddr().String()
	parts := strings.Split(addr, ":")
	port := parts[len(parts)-1]
	if err := pc.Close(); err != nil {
		slog.Error("failed to close packet connection", "error", err)
	}

	mock := &MockDNSServer{
		Address: "127.0.0.1",
		Port:    port,
	}

	// Set up DNS handler
	dns.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)

		// Simple response for testing
		if len(r.Question) > 0 {
			q := r.Question[0]
			if q.Qtype == dns.TypeA {
				rr, _ := dns.NewRR(fmt.Sprintf("%s A 1.2.3.4", q.Name))
				m.Answer = append(m.Answer, rr)
			}
		}

		_ = w.WriteMsg(m)
	})

	// Start server
	server := &dns.Server{Addr: fmt.Sprintf("127.0.0.1:%s", port), Net: "udp"}
	mock.server = server

	go server.ListenAndServe()

	return mock, nil
}

// Close shuts down the mock DNS server
func (m *MockDNSServer) Close() {
	if m.server != nil {
		_ = m.server.Shutdown()
	}
}

// MockSSHServer creates a mock SSH server for testing
// Note: Simplified version - full implementation would require crypto/ssh
type MockSSHServer struct {
	Address string
	Port    string
	listener net.Listener
}

// NewMockSSHServer creates a new mock SSH server
func NewMockSSHServer() (*MockSSHServer, error) {
	// Listen on random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	addr := listener.Addr().String()
	parts := strings.Split(addr, ":")
	port := parts[len(parts)-1]

	mock := &MockSSHServer{
		Address: "127.0.0.1",
		Port:    port,
		listener: listener,
	}

	// Accept connections (simplified - just accept and close)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			// Send SSH banner
			if _, err := conn.Write([]byte("SSH-2.0-MockSSH\r\n")); err != nil {
				slog.Error("failed to write SSH banner", "error", err)
			}
			if err := conn.Close(); err != nil {
				slog.Error("failed to close mock SSH connection", "error", err)
			}
		}
	}()

	return mock, nil
}

// Close shuts down the mock SSH server
func (m *MockSSHServer) Close() {
	if m.listener != nil {
		if err := m.listener.Close(); err != nil {
			slog.Error("failed to close mock SSH listener", "error", err)
		}
	}
}

// MockSMTPServer creates a mock SMTP server for testing
type MockSMTPServer struct {
	Address string
	Port    string
	listener net.Listener
}

// NewMockSMTPServer creates a new mock SMTP server
func NewMockSMTPServer() (*MockSMTPServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	addr := listener.Addr().String()
	parts := strings.Split(addr, ":")
	port := parts[len(parts)-1]

	mock := &MockSMTPServer{
		Address: "127.0.0.1",
		Port:    port,
		listener: listener,
	}

	// Accept connections
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			// Send SMTP banner
			if _, err := conn.Write([]byte("220 MockSMTP Service ready\r\n")); err != nil {
				slog.Error("failed to write SMTP banner", "error", err)
			}
			if err := conn.Close(); err != nil {
				slog.Error("failed to close mock SMTP connection", "error", err)
			}
		}
	}()

	return mock, nil
}

// Close shuts down the mock SMTP server
func (m *MockSMTPServer) Close() {
	if m.listener != nil {
		if err := m.listener.Close(); err != nil {
			slog.Error("failed to close mock SMTP listener", "error", err)
		}
	}
}

// AllMockServers holds all mock servers for easy management
type AllMockServers struct {
	Web  *MockWebServer
	DNS  *MockDNSServer
	SSH  *MockSSHServer
	SMTP *MockSMTPServer
}

// StartAllMockServers starts all mock servers
func StartAllMockServers() (*AllMockServers, error) {
	web := NewMockWebServer("<html><body>Test Page</body></html>")

	dns, err := NewMockDNSServer()
	if err != nil {
		web.Close()
		return nil, err
	}

	ssh, err := NewMockSSHServer()
	if err != nil {
		web.Close()
		dns.Close()
		return nil, err
	}

	smtp, err := NewMockSMTPServer()
	if err != nil {
		web.Close()
		dns.Close()
		ssh.Close()
		return nil, err
	}

	return &AllMockServers{
		Web:  web,
		DNS:  dns,
		SSH:  ssh,
		SMTP: smtp,
	}, nil
}

// CloseAll shuts down all mock servers
func (a *AllMockServers) CloseAll() {
	if a.Web != nil {
		a.Web.Close()
	}
	if a.DNS != nil {
		a.DNS.Close()
	}
	if a.SSH != nil {
		a.SSH.Close()
	}
	if a.SMTP != nil {
		a.SMTP.Close()
	}
}
