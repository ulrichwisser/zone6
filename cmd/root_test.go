package cmd

import (
	"testing"
)

func TestFomatResolver(t *testing.T) {
	valid := []string{
		"1.2.3.4",
		"1.2.3.4:53",
		"1.2.3.4:",
		"1::2",
		"[1::2]:53",
		"[1::2]:",
		"[1::2%eth0]:53",
		"[1::2%eth0]:",
		"[1:2:3:4:5:6:7:8]:54",
	}

	invalid := []string{
		"[1:2:3:4:5:6:7:8:9]:53",
		"[1:2:3:4:5:6:7:8]:70000",
		"1::2%eth0",
	}
	for _, addr := range valid {
		if formatResolver(addr) == "" {
			t.Errorf("Addr %s should be accepted", addr)
		}
	}
	for _, addr := range invalid {
		if newr := formatResolver(addr); newr != "" {
			t.Errorf("Addr %s id false accepted as %s", addr, newr)
		}
	}
}

func TestFomatResolvers(t *testing.T) {
	resolvers := []string{
		// valid
		"1.2.3.4",
		"1.2.3.4:53",
		"1.2.3.4:",
		"1::2",
		"[1::2]:53",
		"[1::2]:",
		"[1::2%eth0]:53",
		"[1::2%eth0]:",
		"[1:2:3:4:5:6:7:8]:54",

		// invalid
		"[1:2:3:4:5:6:7:8:9]:53",
		"[1:2:3:4:5:6:7:8]:70000",
		"1::2%eth0",
	}

	newresolvers := formatResolvers(resolvers)
	if len(newresolvers) != len(resolvers)-3 {
		t.Errorf("formatResolvers has accepted %d resolvers (expected: %d)", len(newresolvers), len(resolvers)-3)
	}
}
