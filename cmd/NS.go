/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"log"
	"sync"

	"github.com/miekg/dns"
	"github.com/spf13/cobra"
)

const TIMEOUT = 5

// store results and manage access
var domain2host map[string][]string = make(map[string][]string)
var hostIPv6 map[string]bool = make(map[string]bool)
var access sync.Mutex

// keep processing stats
var RRinZonefile int = 0
var NSinZonefile int = 0
var ResolvErrors int = 0

// NSCmd represents the NS command
var NSCmd = &cobra.Command{
	Use:   "NS",
	Short: "Statistics about hosts and domain with IPv6",
	Long: `Extracts NS records from zone file and tries to resolve all hosts.
	Computes number of hosts with IPv6 and number of domains with IPv6 DNS.
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 || args[0] == "" {
			log.Fatal("Zone file must be given")
		}

		// get and sync channel
		rrlist := GetZone(args[0], "")

		//
		if verbose > 0 {
			log.Printf("Starting resolving zone file %s\n", args[0])
			log.Printf("Using resolvers: %v\n", resolvers)
		}

		// handle concurrency
		var wg sync.WaitGroup
		var threads = make(chan string, concurrent)
		defer close(threads)

		// rotate between all resolvers
		resolver := 0

		for rr := range rrlist {
			RRinZonefile++
			if rr.Header().Rrtype == dns.TypeNS {
				NSinZonefile++
				domain := rr.Header().Name
				host := rr.(*dns.NS).Ns
				if verbose > 2 {
					log.Printf("%s NS %s", domain, host)
				}
				addNS(domain, host)
				if setHostIfUndefined(host) {
					if verbose > 2 {
						log.Printf("Start resolving host %s", host)
					}
					wg.Add(1)
					threads <- "x"
					go resolv(host, resolvers[resolver], &wg, threads)
					resolver = (resolver + 1) % len(resolvers)
				}
			}
		}

		// done with zone file, wait for all resolvers to finish
		wg.Wait()

		// Processing stats
		log.Println("")
		log.Println("")
		log.Printf("RR processed           %7d", RRinZonefile)
		log.Printf("NS processed           %7d", NSinZonefile)
		log.Printf("Reslove errors         %7d", ResolvErrors)
		log.Printf("Domains found          %7d", len(domain2host))
		log.Printf("Hosts found            %7d", len(hostIPv6))
		log.Println("")

		// host stats
		var hasV6 int = 0
		for host := range hostIPv6 {
			if hostIPv6[host] {
				hasV6++
			}
		}
		log.Printf("Hosts with   IPv6      %7d", hasV6)
		log.Printf("Host without IPv6      %7d", len(hostIPv6)-hasV6)
		log.Println("")

		// statistics counter
		var stats map[int]int = make(map[int]int)

		const (
			hasAllV6 = iota
			hasSomeV6
			hasNoV6
		)

		// compute
		for domain := range domain2host {
			c := 0
			for _, host := range domain2host[domain] {
				if hostIPv6[host] {
					c++
				}
			}
			switch {
			case c == len(domain2host[domain]):
				stats[hasAllV6]++
			case c > 0:
				stats[hasSomeV6]++
			case c == 0:
				stats[hasNoV6]++
			default:
				log.Fatal("How did we get here???")
			}
		}

		// print results
		log.Printf("Domains with all  IPv6 %7d", stats[hasAllV6])
		log.Printf("Domains with some IPv6 %7d", stats[hasSomeV6])
		log.Printf("Domains without   IPv6 %7d", stats[hasNoV6])
	},
}

func init() {
	rootCmd.AddCommand(NSCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// NSCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// NSCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// resolv will send a query and return the result
func resolv(domain string, server string, wg *sync.WaitGroup, threads <-chan string) {
	if verbose > 1 {
		fmt.Printf("Resolving %s using %s\n", domain, server)
	}

	defer func() { _ = <-threads }()
	defer wg.Done()

	// Setting up query
	query := new(dns.Msg)
	query.RecursionDesired = true
	query.Question = make([]dns.Question, 1)
	query.SetQuestion(domain, dns.TypeAAAA)

	// Setting up resolver
	client := new(dns.Client)
	client.ReadTimeout = TIMEOUT * 1e9

	// make the query and wait for answer
	r, _, err := client.Exchange(query, server)

	// check for errors
	if err != nil {
		log.Printf("%-30s: Error resolving %s (server %s)\n", domain, err, server)
		ResolvErrors++
		return
	}
	if r == nil {
		log.Printf("%-30s: No answer (Server %s)\n", domain, server)
		ResolvErrors++
		return
	}
	if r.Rcode != dns.RcodeSuccess {
		log.Printf("%-30s: %s (Rcode %d, Server %s)\n", domain, dns.RcodeToString[r.Rcode], r.Rcode, server)
		ResolvErrors++
		return
	}

	// print out all NS
	for _, answer := range r.Answer {
		if answer.Header().Rrtype == dns.TypeAAAA {
			ipv6 := answer.(*dns.AAAA).AAAA.String()
			if verbose > 2 {
				fmt.Printf("%s %s\n", domain, ipv6)
			}
			setHostHasIPv6(domain)
		}
	}
}

func addNS(domain, host string) {
	if _, ok := domain2host[domain]; !ok {
		domain2host[domain] = make([]string, 0)
	}
	for _, h := range domain2host[domain] {
		if h == host {
			return
		}
	}
	domain2host[domain] = append(domain2host[domain], host)
}

func setHostIfUndefined(host string) bool {
	access.Lock()
	defer access.Unlock()
	if _, ok := hostIPv6[host]; !ok {
		hostIPv6[host] = false
		return true
	}
	return false
}

func setHostHasIPv6(host string) {
	access.Lock()
	defer access.Unlock()
	if _, ok := hostIPv6[host]; !ok {
		log.Fatalf("Host %s has no value", host)
	}
	hostIPv6[host] = true
}
