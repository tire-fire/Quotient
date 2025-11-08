package checks

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
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

func (c Dns) Run(teamID uint, teamIdentifier string, roundID uint, resultsChan chan Result) {
	definition := func(teamID uint, teamIdentifier string, checkResult Result, response chan Result) {
		// Pick a record
		record := c.Record[rand.Intn(len(c.Record))]
		fqdn := dns.Fqdn(strings.ReplaceAll(dns.Fqdn(record.Domain), "_", teamIdentifier))

		// Setup for dns query
		var msg dns.Msg

		// switch of kind of record (A, MX, etc)
		switch record.Kind {
		case "A":
			msg.SetQuestion(fqdn, dns.TypeA)
		case "AAAA":
			msg.SetQuestion(fqdn, dns.TypeAAAA)
		case "CNAME":
			msg.SetQuestion(fqdn, dns.TypeCNAME)
		case "MX":
			msg.SetQuestion(fqdn, dns.TypeMX)
		case "NS":
			msg.SetQuestion(fqdn, dns.TypeNS)
		case "PTR":
			msg.SetQuestion(fqdn, dns.TypePTR)
		case "SOA":
			msg.SetQuestion(fqdn, dns.TypeSOA)
		case "SRV":
			msg.SetQuestion(fqdn, dns.TypeSRV)
		case "TXT":
			msg.SetQuestion(fqdn, dns.TypeTXT)
		}

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

		// Loop through results and check for correct match
		for _, answer := range in.Answer {
			// Check the answer based on record type
			for _, expectedAnswer := range record.Answer {
				expectedAnswer = strings.ReplaceAll(expectedAnswer, "_", teamIdentifier)
				var actualAnswer string

				switch record.Kind {
				case "A":
					if a, ok := answer.(*dns.A); ok {
						actualAnswer = a.A.String()
					}
				case "AAAA":
					if aaaa, ok := answer.(*dns.AAAA); ok {
						actualAnswer = aaaa.AAAA.String()
					}
				case "CNAME":
					if cname, ok := answer.(*dns.CNAME); ok {
						actualAnswer = cname.Target
					}
				case "MX":
					if mx, ok := answer.(*dns.MX); ok {
						actualAnswer = mx.Mx
					}
				case "NS":
					if ns, ok := answer.(*dns.NS); ok {
						actualAnswer = ns.Ns
					}
				case "PTR":
					if ptr, ok := answer.(*dns.PTR); ok {
						actualAnswer = ptr.Ptr
					}
				case "SOA":
					if soa, ok := answer.(*dns.SOA); ok {
						actualAnswer = soa.Ns
					}
				case "SRV":
					if srv, ok := answer.(*dns.SRV); ok {
						actualAnswer = srv.Target
					}
				case "TXT":
					if txt, ok := answer.(*dns.TXT); ok {
						actualAnswer = strings.Join(txt.Txt, " ")
					}
				}

				if actualAnswer == expectedAnswer {
					checkResult.Status = true
					checkResult.Debug = fmt.Sprintf("record %s returned %s. acceptable answers were: %v", record.Domain, expectedAnswer, record.Answer)
					response <- checkResult
					return
				}
			}
		}

		// If we reach here no records matched expected IP and check fails
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
	if err := c.Service.Configure(ip, points, timeout, slapenalty, slathreshold); err != nil {
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

	return nil
}
