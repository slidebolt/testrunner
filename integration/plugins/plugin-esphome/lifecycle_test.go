package pluginesphome

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/slidebolt/sdk-types"
	"github.com/slidebolt/testrunner/integration/testutil"
)

func TestESPHomeLifecycle(t *testing.T) {
	const pluginID = "plugin-esphome"
	testutil.RequirePlugin(t, pluginID)

	t.Run("Device Listing", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/plugins/%s/devices", testutil.APIBaseURL(), pluginID)
		resp, err := (&http.Client{Timeout: 2 * time.Second}).Get(url)
		if err != nil {
			t.Fatalf("Failed to fetch devices: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		var devices []types.Device
		if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
			t.Fatalf("Failed to decode devices: %v", err)
		}

		t.Logf("Plugin reported %d devices", len(devices))
		
		for _, dev := range devices {
			t.Run(fmt.Sprintf("Entity Listing for %s", dev.ID), func(t *testing.T) {
				entUrl := fmt.Sprintf("%s/api/plugins/%s/devices/%s/entities", testutil.APIBaseURL(), pluginID, dev.ID)
				entResp, err := (&http.Client{Timeout: 2 * time.Second}).Get(entUrl)
				if err != nil {
					t.Fatalf("Failed to fetch entities for %s: %v", dev.ID, err)
				}
				defer entResp.Body.Close()

				if entResp.StatusCode != http.StatusOK {
					t.Fatalf("Expected status 200 for entities, got %d", entResp.StatusCode)
				}

				var entities []types.Entity
				if err := json.NewDecoder(entResp.Body).Decode(&entities); err != nil {
					t.Fatalf("Failed to decode entities: %v", err)
				}
				t.Logf("Device %s has %d entities", dev.ID, len(entities))
			})
		}
	})
}
