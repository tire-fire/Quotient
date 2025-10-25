package checks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"al.essio.dev/pkg/shellescape"
)

type Custom struct {
	Service
	Command string
	Regex   string
}

func (c Custom) Run(teamID uint, teamIdentifier string, roundID uint, resultsChan chan Result) {
	// Prepare command substitution early so it's available for timeout debugging
	var username, password string
	var formedCommand string
	var credErr error

	if len(c.CredLists) > 0 {
		username, password, credErr = c.getCreds(teamID)
	}

	// Replace command input keywords
	formedCommand = c.Command
	formedCommand = strings.Replace(formedCommand, "ROUND", strconv.FormatUint(uint64(roundID), 10), -1)

	// Replace underscore in target with team identifier before substitution
	target := strings.Replace(c.Target, "_", teamIdentifier, -1)
	formedCommand = strings.Replace(formedCommand, "TARGET", target, -1)
	formedCommand = strings.Replace(formedCommand, "TEAMIDENTIFIER", teamIdentifier, -1)

	// Shell escape username and password
	if len(c.CredLists) > 0 {
		formedCommand = strings.Replace(formedCommand, "USERNAME", shellescape.Quote(username), -1)
		formedCommand = strings.Replace(formedCommand, "PASSWORD", shellescape.Quote(password), -1)
	}

	definition := func(teamID uint, teamIdentifier string, checkResult Result, response chan Result) {
		// Set formed command in debug immediately so it's available on timeout
		checkResult.Debug = formedCommand

		if credErr != nil {
			checkResult.Error = "error getting creds"
			checkResult.Debug = credErr.Error()
			response <- checkResult
			return
		}

		slog.Debug("CUSTOM CHECK COMMAND", "command", formedCommand)

		// Create command with timeout context
		timeout := time.Duration(c.Timeout) * time.Second
		if timeout == 0 {
			timeout = 30 * time.Second // fallback if timeout not set
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, "/bin/sh", "-c", formedCommand)

		tmpfilePath := fmt.Sprintf("/tmp/custom-check-%d-%d-%s", roundID, teamID, c.Name)
		tmpfile, err := os.Create(tmpfilePath)
		if err != nil {
			checkResult.Error = "error creating tmpfile"
			checkResult.Debug = err.Error()
			response <- checkResult
			return
		}
		defer tmpfile.Close()

		cmd.Stdout = tmpfile
		cmd.Stderr = tmpfile

		err = cmd.Run()
		raw, err2 := os.ReadFile(tmpfilePath)
		if err2 != nil {
			checkResult.Debug += fmt.Sprintf("\nerror reading tmpfile:\n%s", err2.Error())
			response <- checkResult
			return
		}
		out := string(raw)
		if err != nil {
			checkResult.Error += fmt.Sprintf("command returned error:\n%s", err.Error())
			checkResult.Debug += fmt.Sprintf("\noutput:\n%s", out)
			response <- checkResult
			return
		}
		if c.Regex != "" {
			re, err := regexp.Compile(c.Regex)
			if err != nil {
				checkResult.Error = "error compiling regex"
				checkResult.Debug = err.Error()
				response <- checkResult
				return
			}

			reFind := re.Find([]byte(out))
			if reFind == nil {
				checkResult.Error = "output incorrect"
				checkResult.Debug += " couldn't find regex \"" + c.Regex + "\" in " + out
				response <- checkResult
				return
			}
			checkResult.Status = true
			checkResult.Debug += " found regex \"" + c.Regex + "\" in " + out
			response <- checkResult
			return
		}

		checkResult.Status = true
		checkResult.Debug += " " + out
		response <- checkResult
	}

	c.Service.Run(teamID, teamIdentifier, roundID, resultsChan, definition)
}

func (c *Custom) Verify(box string, ip string, points int, timeout int, slapenalty int, slathreshold int) error {
	if c.ServiceType == "" {
		c.ServiceType = "Custom"
	}
	if err := c.Service.Configure(ip, points, timeout, slapenalty, slathreshold); err != nil {
		return err
	}
	if c.Display == "" {
		c.Display = "custom"
	}
	if c.Name == "" {
		c.Name = box + "-" + c.Display
	}
	if c.Command == "" {
		return errors.New("no command found for custom check " + c.Name)
	}

	return nil
}
