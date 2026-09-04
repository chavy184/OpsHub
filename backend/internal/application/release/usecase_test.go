package release

import (
	"testing"

	"ops-hub/internal/domain/host"
)

func TestMatchesTargetHostRequiresExactAddressMatch(t *testing.T) {
	prodHost := &host.Host{
		ID:          "host-1",
		Name:        "prod-db",
		HostAddress: "10.0.0.1",
	}

	if matchesTargetHost([]string{"10.0.0.10"}, prodHost) {
		t.Fatal("expected partial IP address not to match prod host")
	}

	if !matchesTargetHost([]string{"10.0.0.1"}, prodHost) {
		t.Fatal("expected exact IP address to match prod host")
	}
}
