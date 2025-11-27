package testutil

import (
	"fmt"
)

// TeamFixture represents test team data
type TeamFixture struct {
	ID         uint
	Name       string
	Identifier string
	Password   string
}

// BoxFixture represents test box data
type BoxFixture struct {
	Name string
	IP   string
}

// ServiceFixture represents test service data
type ServiceFixture struct {
	Type    string
	Name    string
	Port    int
	Enabled bool
}

// GetTestTeams returns a set of test teams
func GetTestTeams() []TeamFixture {
	return []TeamFixture{
		{ID: 1, Name: "Team01", Identifier: "01", Password: "password1"},
		{ID: 2, Name: "Team02", Identifier: "02", Password: "password2"},
		{ID: 3, Name: "Team03", Identifier: "03", Password: "password3"},
	}
}

// GetTestBoxes returns a set of test boxes
func GetTestBoxes() []BoxFixture {
	return []BoxFixture{
		{Name: "web01", IP: "10.100.1_.2"},
		{Name: "mail01", IP: "10.100.1_.3"},
		{Name: "dns01", IP: "10.100.1_.4"},
	}
}

// GetTestServices returns a set of test services
func GetTestServices() []ServiceFixture {
	return []ServiceFixture{
		{Type: "Web", Name: "web01-web", Port: 80, Enabled: true},
		{Type: "SSH", Name: "web01-ssh", Port: 22, Enabled: true},
		{Type: "SMTP", Name: "mail01-smtp", Port: 25, Enabled: true},
		{Type: "DNS", Name: "dns01-dns", Port: 53, Enabled: true},
	}
}

// SubstituteTeamIdentifier replaces underscore with team identifier in IP
func SubstituteTeamIdentifier(ip, identifier string) string {
	result := ""
	for _, ch := range ip {
		if ch == '_' {
			result += identifier
		} else {
			result += string(ch)
		}
	}
	return result
}

// GetTeamBoxIP returns the IP for a team's box
func GetTeamBoxIP(box BoxFixture, team TeamFixture) string {
	return SubstituteTeamIdentifier(box.IP, team.Identifier)
}

// CreateTestConfig generates a test TOML configuration
func CreateTestConfig(teams []TeamFixture, boxes []BoxFixture) string {
	config := `[RequiredSettings]
EventName = "Test Event"
EventType = "rvb"
DBConnectURL = "postgres://postgres:postgres@localhost:5432/quotient_test?sslmode=disable"
BindAddress = "127.0.0.1"

[MiscSettings]
Delay = 60
Jitter = 5
Points = 5
Timeout = 10
StartPaused = true
`

	// Add teams
	for _, team := range teams {
		config += fmt.Sprintf(`
[[team]]
name = "%s"
pw = "%s"
`, team.Name, team.Password)
	}

	// Add boxes with services
	for _, box := range boxes {
		config += fmt.Sprintf(`
[[box]]
name = "%s"
ip = "%s"
`, box.Name, box.IP)

		// Add services based on box type
		if box.Name == "web01" {
			config += `
  [[box.web]]
  display = "Web Service"
  port = 80

  [[box.ssh]]
  display = "SSH Service"
  port = 22
`
		} else if box.Name == "mail01" {
			config += `
  [[box.smtp]]
  display = "SMTP Service"
  port = 25
`
		} else if box.Name == "dns01" {
			config += `
  [[box.dns]]
  display = "DNS Service"
  port = 53
  record = "test.example.com"
  recordtype = "A"
  answer = "1.2.3.4"
`
		}
	}

	return config
}

// TaskFixture represents test task data
type TaskFixture struct {
	TeamID        uint
	TeamIdentifier string
	ServiceType    string
	ServiceName    string
	RoundID        uint
}

// GetTestTasks creates sample tasks for testing
func GetTestTasks() []TaskFixture {
	return []TaskFixture{
		{TeamID: 1, TeamIdentifier: "01", ServiceType: "Web", ServiceName: "web01-web", RoundID: 1},
		{TeamID: 1, TeamIdentifier: "01", ServiceType: "SSH", ServiceName: "web01-ssh", RoundID: 1},
		{TeamID: 2, TeamIdentifier: "02", ServiceType: "Web", ServiceName: "web01-web", RoundID: 1},
		{TeamID: 2, TeamIdentifier: "02", ServiceType: "SSH", ServiceName: "web01-ssh", RoundID: 1},
	}
}

// ResultFixture represents test result data
type ResultFixture struct {
	TeamID      uint
	ServiceName string
	ServiceType string
	RoundID     uint
	Status      bool
	Points      int
}

// GetTestResults creates sample results for testing
func GetTestResults() []ResultFixture {
	return []ResultFixture{
		{TeamID: 1, ServiceName: "web01-web", ServiceType: "Web", RoundID: 1, Status: true, Points: 5},
		{TeamID: 1, ServiceName: "web01-ssh", ServiceType: "SSH", RoundID: 1, Status: true, Points: 5},
		{TeamID: 2, ServiceName: "web01-web", ServiceType: "Web", RoundID: 1, Status: false, Points: 0},
		{TeamID: 2, ServiceName: "web01-ssh", ServiceType: "SSH", RoundID: 1, Status: true, Points: 5},
	}
}
