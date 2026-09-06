package writefreely

import (
	"net"
	"path/filepath"
	"strings"
	"testing"

	"github.com/writefreely/writefreely/config"
)

func TestUpstreamSSRFAddressGuards(t *testing.T) {
	for _, tc := range []struct {
		address string
		public  bool
	}{
		{"127.0.0.1", false},
		{"10.0.0.1", false},
		{"169.254.169.254", false},
		{"100.64.0.1", false},
		{"224.0.0.1", false},
		{"8.8.8.8", true},
		{"2001:4860:4860::8888", true},
	} {
		t.Run(tc.address, func(t *testing.T) {
			if got := isPublicAddr(net.ParseIP(tc.address)); got != tc.public {
				t.Fatalf("isPublicAddr(%q) = %t, want %t", tc.address, got, tc.public)
			}
		})
	}

	if err := isPublicIRI("https://127.0.0.1/actor"); err == nil {
		t.Fatal("ActivityPub IRI guard accepted a loopback address")
	}
	if err := isPublicIRI("ftp://8.8.8.8/actor"); err == nil {
		t.Fatal("ActivityPub IRI guard accepted a non-HTTP scheme")
	}
	if err := isPublicIRI("https://8.8.8.8/actor"); err != nil {
		t.Fatalf("ActivityPub IRI guard rejected a public address: %v", err)
	}
}

func TestUpstreamMissingTemplatesError(t *testing.T) {
	cfg := config.New()
	cfg.Server.TemplatesParentDir = filepath.Join(t.TempDir(), "missing")
	err := InitTemplates(cfg)
	if err == nil || !strings.Contains(err.Error(), "unable to find templates directory") {
		t.Fatalf("InitTemplates error = %v", err)
	}
}
