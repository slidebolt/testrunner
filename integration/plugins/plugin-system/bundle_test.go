package pluginsystem

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/slidebolt/sdk-types"
	"github.com/slidebolt/testrunner/integration/testutil"
)

func TestSystemEventFlow(t *testing.T) {
	pluginID := "plugin-system"
	testutil.RequirePlugin(t, pluginID)

	t.Run("Event Reception", func(t *testing.T) {
		// Wait for at least one sensor update to prove data is moving
		deviceID := "system-device"
		entityID := "system-cpu"
		deadline := time.Now().Add(10 * time.Second)
		url := fmt.Sprintf("%s/api/plugins/%s/devices/%s/entities", testutil.APIBaseURL(), pluginID, deviceID)

		success := false
		for time.Now().Before(deadline) {
			resp, err := http.Get(url)
			if err == nil {
				var entities []types.Entity
				if err := json.NewDecoder(resp.Body).Decode(&entities); err == nil {
					resp.Body.Close()
					for _, ent := range entities {
						if ent.ID == entityID && len(ent.Data.Reported) > 0 {
							var state struct {
								Usage float64 `json:"percent"`
								TS    string  `json:"ts"`
							}
							if err := json.Unmarshal(ent.Data.Reported, &state); err == nil {
								if state.TS != "" {
									success = true
									t.Logf("Received CPU event: usage=%v, ts=%v", state.Usage, state.TS)
									break
								}
							}
						}
					}
				} else {
					resp.Body.Close()
				}
			}
			if success {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}

		if !success {
			t.Errorf("failed to receive sensor events within timeout")
		}
	})
}
