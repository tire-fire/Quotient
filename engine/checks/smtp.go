package checks

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// safeFortunes contains a curated list of appropriate fortunes for email content.
// These replace the system fortune database which contains inappropriate material.
var safeFortunes = []string{
	"The best way to predict the future is to invent it. - Alan Kay",
	"Simplicity is the ultimate sophistication. - Leonardo da Vinci",
	"First, solve the problem. Then, write the code. - John Johnson",
	"Code is like humor. When you have to explain it, it's bad. - Cory House",
	"Make it work, make it right, make it fast. - Kent Beck",
	"The only way to do great work is to love what you do. - Steve Jobs",
	"Talk is cheap. Show me the code. - Linus Torvalds",
	"Programs must be written for people to read. - Harold Abelson",
	"Any fool can write code that a computer can understand. Good programmers write code that humans can understand. - Martin Fowler",
	"The best error message is the one that never shows up. - Thomas Fuchs",
	"Debugging is twice as hard as writing the code in the first place. - Brian Kernighan",
	"Perfection is achieved not when there is nothing more to add, but when there is nothing left to take away. - Antoine de Saint-Exupery",
	"Java is to JavaScript what car is to carpet. - Chris Heilmann",
	"Knowledge is power. - Francis Bacon",
	"In theory, theory and practice are the same. In practice, they are not. - Albert Einstein",
	"The computer was born to solve problems that did not exist before. - Bill Gates",
	"A good programmer looks both ways before crossing a one-way street.",
	"Weeks of coding can save you hours of planning.",
	"It works on my machine.",
	"There are only two hard things in computer science: cache invalidation and naming things. - Phil Karlton",
	"The best thing about a boolean is even if you are wrong, you are only off by a bit.",
	"A user interface is like a joke. If you have to explain it, it's not that good.",
	"Computers are fast; programmers keep them slow.",
	"Copy and paste is a design error. - David Parnas",
	"Deleted code is debugged code. - Jeff Sickel",
	"If debugging is the process of removing bugs, then programming must be the process of putting them in. - Edsger Dijkstra",
	"The most disastrous thing that you can ever learn is your first programming language. - Alan Kay",
	"One man's crappy software is another man's full-time job. - Jessica Gaston",
	"Always code as if the guy who ends up maintaining your code will be a violent psychopath who knows where you live. - John Woods",
	"Programming is the art of telling another human being what one wants the computer to do. - Donald Knuth",
}

type Smtp struct {
	Service
	Encrypted   bool
	Domain      string
	RequireAuth bool
	Fortunes    []string
}

type unencryptedAuth struct {
	smtp.Auth
}

func (a unencryptedAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	s := *server
	s.TLS = true

	return a.Auth.Start(&s)
}

func (c Smtp) Run(teamID uint, teamIdentifier string, roundID uint, resultsChan chan Result) {
	definition := func(teamID uint, teamIdentifier string, checkResult Result, response chan Result) {
		c.Fortunes = safeFortunes

		// Create a dialer
		dialer := net.Dialer{
			Timeout: time.Duration(c.Timeout) * time.Second,
		}

		fortune := c.Fortunes[rand.Intn(len(c.Fortunes))] // #nosec G404 -- non-crypto selection of fortune text
		words := strings.Fields(fortune)
		subject := ""
		if len(words) <= 3 {
			subject = fortune
		} else {
			selected := make([]string, 3)
			for i := range 3 {
				selected[i] = words[rand.Intn(len(words))] // #nosec G404 -- non-crypto selection of words for subject
			}
			subject = strings.Join(selected, " ")
		}

		// ***********************************************
		// Set up custom auth for bypassing net/smtp protections
		username, password, err := c.getCreds(teamID)
		if err != nil {
			checkResult.Error = "error getting creds"
			checkResult.Debug = err.Error()
			response <- checkResult
			return
		}

		toUser, _, err := c.getCreds(teamID)
		if err != nil {
			checkResult.Error = "error getting creds"
			checkResult.Debug = err.Error()
			response <- checkResult
			return
		}

		auth := unencryptedAuth{smtp.PlainAuth("", username+c.Domain, password, c.Target)}
		// ***********************************************

		if c.Domain != "" {
			username = username + c.Domain
			toUser = toUser + c.Domain
		}

		// The good way to do auth
		// auth := smtp.PlainAuth("", d.Username, d.Password, d.Host)
		// Create TLS config
		tlsConfig := tls.Config{
			InsecureSkipVerify: true, // #nosec G402 -- competition services may use self-signed certs
		}

		// Declare these for the below if block
		var conn net.Conn

		if c.Encrypted {
			conn, err = tls.DialWithDialer(&dialer, "tcp", fmt.Sprintf("%s:%d", c.Target, c.Port), &tlsConfig)
		} else {
			conn, err = dialer.DialContext(context.TODO(), "tcp", fmt.Sprintf("%s:%d", c.Target, c.Port))
		}
		if err != nil {
			checkResult.Error = "connection to server failed"
			checkResult.Debug = err.Error()
			response <- checkResult
			return
		}
		defer func() {
		if err := conn.Close(); err != nil {
			slog.Error("failed to close smtp connection", "error", err)
		}
	}()

		// Create smtp client
		sconn, err := smtp.NewClient(conn, c.Target)
		if err != nil {
			checkResult.Error = "smtp client creation failed"
			checkResult.Debug = err.Error()
			response <- checkResult
			return
		}
		defer sconn.Quit()

		// Login
		if len(c.CredLists) > 0 {
			authSupported, _ := sconn.Extension("AUTH")
			if c.RequireAuth || authSupported {
				err = sconn.Auth(auth)
				if err != nil {
					checkResult.Error = "login failed for " + username + ":" + password
					checkResult.Debug = err.Error()
					response <- checkResult
					return
				}
			}
		}

		// Set the sender
		err = sconn.Mail(username)
		if err != nil {
			checkResult.Error = "setting sender failed"
			checkResult.Debug = err.Error()
			response <- checkResult
			return
		}

		// Set the receiver
		err = sconn.Rcpt(toUser)
		if err != nil {
			checkResult.Error = "setting receiver failed"
			checkResult.Debug = err.Error()
			response <- checkResult
			return
		}

		// Create email writer
		wc, err := sconn.Data()
		if err != nil {
			checkResult.Error = "creating email writer failed"
			checkResult.Debug = err.Error()
			response <- checkResult
			return
		}
		defer func() {
			if err := wc.Close(); err != nil {
				slog.Error("failed to close smtp writer", "error", err)
			}
		}()

		body := fmt.Sprintf("Subject: %s\n\n%s\n\n", subject, fortune)

		// Write the body using Fprint to avoid treating the contents as a
		// format string.
		_, err = fmt.Fprint(wc, body)
		if err != nil {
			checkResult.Error = "writing body failed"
			checkResult.Debug = err.Error()
			response <- checkResult
			return
		}

		checkResult.Status = true
		checkResult.Debug = "successfully wrote '" + body + "' to " + toUser + " from " + username
		response <- checkResult
	}

	c.Service.Run(teamID, teamIdentifier, roundID, resultsChan, definition)
}

func (c *Smtp) Verify(box string, ip string, points int, timeout int, slapenalty int, slathreshold int) error {
	if c.ServiceType == "" {
		c.ServiceType = "Smtp"
	}
	if err := c.Service.Configure(ip, points, timeout, slapenalty, slathreshold); err != nil {
		return err
	}
	if c.Display == "" {
		c.Display = "smtp"
	}
	if c.Name == "" {
		c.Name = box + "-" + c.Display
	}
	if c.Port == 0 {
		c.Port = 25
	}

	return nil
}
