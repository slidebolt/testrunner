package pluginfrigate

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/slidebolt/sdk-types"
	"github.com/slidebolt/testrunner/integration/testutil"
)

func TestFrigateRealDiscoveryCount(t *testing.T) {
	const pluginID = "plugin-frigate"
	testutil.RequirePlugin(t, pluginID)

	url := testutil.PluginEnv(pluginID, "FRIGATE_URL", "PLUGIN_FRIGATE_URL", "PLUGIN_FRIGATE_FRIGATE_URL")
	if strings.TrimSpace(url) == "" {
		t.Skip("no Frigate URL configured; skipping real discovery count test")
	}

	client := http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(12 * time.Second)
	lastCount := 0
	for time.Now().Before(deadline) {
		resp, err := client.Get(testutil.APIBaseURL() + "/api/plugins/" + pluginID + "/devices")
		if err == nil && resp.StatusCode == http.StatusOK {
			var devices []types.Device
			if decodeErr := json.NewDecoder(resp.Body).Decode(&devices); decodeErr == nil {
				resp.Body.Close()
				count := 0
				for _, d := range devices {
					if d.ID == "frigate-system" {
						continue
					}
					count++
				}
				lastCount = count
				if count > 0 {
					deviceIDs := make([]string, 0, count)
					for _, d := range devices {
						if d.ID == "frigate-system" {
							continue
						}
						deviceIDs = append(deviceIDs, d.ID)
					}
					sort.Strings(deviceIDs)

					entityRows := make([]string, 0, len(deviceIDs))
					for _, deviceID := range deviceIDs {
						entities, err := listEntities(client, pluginID, deviceID)
						if err != nil {
							entityRows = append(entityRows, fmt.Sprintf("%s: <error: %v>", deviceID, err))
							continue
						}
						parts := make([]string, 0, len(entities))
						for _, e := range entities {
							parts = append(parts, fmt.Sprintf("%s(%s)", e.ID, e.Domain))
						}
						sort.Strings(parts)
						entityRows = append(entityRows, fmt.Sprintf("%s: %s", deviceID, strings.Join(parts, ", ")))
					}

					t.Logf("Frigate discovered %d camera device(s): %s", count, strings.Join(deviceIDs, ", "))
					for _, row := range entityRows {
						t.Logf("entities %s", row)
					}
					return
				}
			} else {
				resp.Body.Close()
			}
		} else if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("Frigate real discovery found %d camera devices within timeout", lastCount)
}

func listEntities(client http.Client, pluginID, deviceID string) ([]types.Entity, error) {
	resp, err := client.Get(testutil.APIBaseURL() + "/api/plugins/" + pluginID + "/devices/" + deviceID + "/entities")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var entities []types.Entity
	if err := json.NewDecoder(resp.Body).Decode(&entities); err != nil {
		return nil, err
	}
	return entities, nil
}
