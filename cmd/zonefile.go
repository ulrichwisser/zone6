package cmd

import (
	"log"
	"os"

	"github.com/miekg/dns"
)

func GetZone(infile string, zone string) <-chan dns.RR {
	// open zone file
	f, err := os.Open(infile)
	if err != nil {
		log.Fatalf("Zone file %s could not be opened", infile)
	}

	// prepare output channel
	out := make(chan dns.RR, 10000)

	// start zone file parsing
	//tokens := dns.ParseZone(f, dns.Fqdn(zone), infile)
	zp := dns.NewZoneParser(f, dns.Fqdn(zone), infile)

	// translate tokens to RR and write to output channel
	go func() {
		defer f.Close()
		defer close(out)
		for {
			rr, ok := zp.Next()
			if rr == nil || !ok {
				break
			}
			out <- rr
		}
		if err := zp.Err(); err != nil {
			log.Fatal(err)
		}
	}()

	// return the output channel
	return out
}
