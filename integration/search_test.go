package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/slidebolt/sdk-types"
)

func TestDistributedSearch(t *testing.T) {
	client := http.Client{}

	t.Run("Plugin Name Search", func(t *testing.T) {
		resp, err := client.Get(apiBaseURL() + "/api/search/plugins?q=*Plugin*")
		if err != nil {
			t.Fatalf("Search request failed: %v", err)
		}
		defer resp.Body.Close()

		var results []types.Manifest
		json.NewDecoder(resp.Body).Decode(&results)
		if len(results) < 2 {
			t.Errorf("Expected at least 2 plugins matching *Plugin*, found %d", len(results))
		}
		fmt.Printf("PASS: Plugin search returned %d results for *Plugin*.\n", len(results))
	})

	t.Run("Device ID Search Across Plugins", func(t *testing.T) {
		// TestDeviceIsolation creates dev-1 through dev-25 for 3 plugins.
		// dev-1* matches dev-1 plus dev-10 through dev-19 = 11 per plugin × 3 = 33.
		resp, err := client.Get(apiBaseURL() + "/api/search/devices?q=dev-1*")
		if err != nil {
			t.Fatalf("Search request failed: %v", err)
		}
		defer resp.Body.Close()

		var results []types.Device
		json.NewDecoder(resp.Body).Decode(&results)
		const expected = 33
		if len(results) != expected {
			t.Errorf("Expected %d devices matching dev-1*, found %d", expected, len(results))
		} else {
			fmt.Printf("PASS: Device search returned %d results for dev-1* across all plugins.\n", len(results))
		}
	})
}
