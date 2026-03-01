package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/slidebolt/sdk-types"
	"github.com/slidebolt/testrunner/integration/testutil"
)

func TestSchemaDomains(t *testing.T) {
	client := http.Client{Timeout: 2 * time.Second}

	t.Run("All Domains Returns Non-Empty List", func(t *testing.T) {
		resp, err := client.Get(testutil.APIBaseURL() + "/api/schema/domains")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var domains []types.DomainDescriptor
		if err := json.NewDecoder(resp.Body).Decode(&domains); err != nil {
			t.Fatalf("failed decoding response: %v", err)
		}
		if len(domains) == 0 {
			t.Fatal("expected at least one domain descriptor, got none")
		}
		fmt.Printf("PASS: Schema endpoint returned %d domain(s).\n", len(domains))
	})

	t.Run("Known Domain Returns Descriptor", func(t *testing.T) {
		resp, err := client.Get(testutil.APIBaseURL() + "/api/schema/domains/switch")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for domain 'switch', got %d", resp.StatusCode)
		}
		var desc types.DomainDescriptor
		if err := json.NewDecoder(resp.Body).Decode(&desc); err != nil {
			t.Fatalf("failed decoding response: %v", err)
		}
		if desc.Domain != "switch" {
			t.Errorf("expected domain 'switch', got %q", desc.Domain)
		}
		fmt.Printf("PASS: Domain 'switch' descriptor returned with %d command(s) and %d event(s).\n", len(desc.Commands), len(desc.Events))
	})

	t.Run("Unknown Domain Returns 404", func(t *testing.T) {
		resp, err := client.Get(testutil.APIBaseURL() + "/api/schema/domains/not-a-real-domain")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404 for unknown domain, got %d", resp.StatusCode)
		}
		fmt.Println("PASS: Unknown domain correctly returns 404.")
	})
}
