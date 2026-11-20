package checks

import (
	"errors"
	"fmt"
	"maps"
	"math/rand"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/miekg/dns"
)

type Dns struct {
	Service
	Record []DnsRecord
}

type DnsRecord struct {
	Kind   string
	Domain string
	Answer []string
}

// dnsKinds matchers accept only answers of their own kind: the team's server chooses the answer type.
var dnsKinds = map[string]struct {
	qtype   uint16
	matches func(answer dns.RR, expected string) bool
}{
	"A": {dns.TypeA, func(answer dns.RR, expected string) bool {
		rr, ok := answer.(*dns.A)
		return ok && rr.A.String() == expected
	}},
	"MX": {dns.TypeMX, func(answer dns.RR, expected string) bool {
		rr, ok := answer.(*dns.MX)
		return ok && strings.EqualFold(rr.Mx, dns.Fqdn(expected))
	}},
}

func (c Dns) Run(teamID uint, teamIdentifier string, roundID uint, resultsChan chan Result) {
	definition := func(teamID uint, teamIdentifier string, checkResult Result, response chan Result) {
		record := c.Record[rand.Intn(len(c.Record))] // #nosec G404 -- non-crypto selection of DNS record to test
		kind := dnsKinds[record.Kind]
		fqdn := dns.Fqdn(strings.ReplaceAll(dns.Fqdn(record.Domain), "_", teamIdentifier))

		var msg dns.Msg
		msg.SetQuestion(fqdn, kind.qtype)

		// Make it obey timeout via deadline
		// deadctx, cancel := context.WithTimeout(context.TODO(), time.Duration(2)*time.Second)
		// defer cancel()

		// Send the query
		client := dns.Client{Timeout: time.Duration(c.Timeout-1) * time.Second, DialTimeout: time.Duration(c.Timeout-1) * time.Second}
		// _, _ = dns.ExchangeContext(deadctx, &msg, fmt.Sprintf("%s:%d", c.Target, c.Port)) // double tap for propagation
		in, rtt, err := client.Exchange(&msg, fmt.Sprintf("%s:%d", c.Target, c.Port))
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				in, rtt, err = client.Exchange(&msg, fmt.Sprintf("%s:%d", c.Target, c.Port))
				if err != nil {
					checkResult.Error = "error sending query"
					checkResult.Debug = "record " + record.Domain + ":" + fmt.Sprint(record.Answer) + fmt.Sprintf("(took %s)", rtt) + ": " + err.Error()
					response <- checkResult
					return
				}
			} else {
				checkResult.Error = "error sending query"
				checkResult.Debug = "record " + record.Domain + ":" + fmt.Sprint(record.Answer) + fmt.Sprintf("(took %s)", rtt) + ": " + err.Error()
				response <- checkResult
				return
			}
		}

		// Check if we got any records
		if len(in.Answer) < 1 {
			checkResult.Error = "no records received"
			checkResult.Debug = "record " + record.Domain + "-> " + fmt.Sprint(record.Answer)
			response <- checkResult
			return
		}

		for _, answer := range in.Answer {
			for _, expectedAnswer := range record.Answer {
				expectedAnswer = strings.ReplaceAll(expectedAnswer, "_", teamIdentifier)
				if kind.matches(answer, expectedAnswer) {
					checkResult.Status = true
					checkResult.Debug = fmt.Sprintf("record %s returned %s. acceptable answers were: %v", record.Domain, expectedAnswer, record.Answer)
					response <- checkResult
					return
				}
			}
		}

		checkResult.Error = "incorrect answer(s) received from DNS"
		checkResult.Debug = "record " + record.Domain + "-> acceptable answers were: " + fmt.Sprint(record.Answer) + ", received " + fmt.Sprint(in.Answer)
		response <- checkResult
	}

	c.Service.Run(teamID, teamIdentifier, roundID, resultsChan, definition)
}

func (c *Dns) Verify(box string, ip string, points int, timeout int, slapenalty int, slathreshold int) error {
	if c.ServiceType == "" {
		c.ServiceType = "Dns"
	}
	if err := c.Configure(ip, points, timeout, slapenalty, slathreshold); err != nil {
		return err
	}
	if c.Port == 0 {
		c.Port = 53
	}
	if len(c.Record) < 1 {
		return errors.New("dns check " + c.Name + " has no records")
	}
	if c.Display == "" {
		c.Display = "dns"
	}
	if c.Name == "" {
		c.Name = box + "-" + c.Display
	}
	for _, record := range c.Record {
		if _, ok := dnsKinds[record.Kind]; !ok {
			return fmt.Errorf("dns check %s: unsupported record kind %q (supported: %s)",
				c.Name, record.Kind, strings.Join(slices.Sorted(maps.Keys(dnsKinds)), ", "))
		}
	}

	return nil
}
