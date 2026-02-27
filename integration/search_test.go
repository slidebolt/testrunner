package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/slidebolt/sdk-types"
	"github.com/slidebolt/testrunner/integration/testutil"
)

func TestDistributedSearch(t *testing.T) {
	client := http.Client{}

	t.Run("Plugin Search Endpoint", func(t *testing.T) {
		resp, err := client.Get(testutil.APIBaseURL() + "/api/search/plugins?q=*")
		if err != nil {
			t.Fatalf("Search request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("unexpected status: %d", resp.StatusCode)
		}

		var results []types.Manifest
		if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
			t.Fatalf("failed decoding response: %v", err)
		}
		fmt.Printf("PASS: Plugin search endpoint returned %d result(s).\n", len(results))
	})

	t.Run("Device Search Endpoint", func(t *testing.T) {
		resp, err := client.Get(testutil.APIBaseURL() + "/api/search/devices?q=*")
		if err != nil {
			t.Fatalf("Search request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("unexpected status: %d", resp.StatusCode)
		}

		var results []types.Device
		if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
			t.Fatalf("failed decoding response: %v", err)
		}
		fmt.Printf("PASS: Device search endpoint returned %d result(s).\n", len(results))
	})
}
