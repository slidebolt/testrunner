package pluginesphome

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/slidebolt/sdk-types"
	"github.com/slidebolt/testrunner/local/testutil"
)

func TestLocalESPHomeRealEntityCoverageRatgdoAndAlexSwitch(t *testing.T) {
	const pluginID = "plugin-esphome"
	testutil.RequirePlugin(t, pluginID)

	url := testutil.PluginEnv(pluginID, "ESPHOME_DASHBOARD_URL", "ESPHOME_URL")
	if strings.TrimSpace(url) == "" {
		t.Skip("no ESPHome dashboard URL configured; skipping real entity coverage test")
	}

	client := http.Client{Timeout: 3 * time.Second}
	devices := waitForDevices(t, client, pluginID, 25*time.Second)

	targets := filterTargetDevices(devices)
	if len(targets) == 0 {
		t.Skip("no ESPHome devices matching ratgdo/alex switch found; skipping target coverage report")
	}

	totalEntities := 0
	domainCounts := map[string]int{}
	for _, d := range targets {
		var cfg struct {
			Address string `json:"address"`
			Key     string `json:"key"`
		}
		_ = json.Unmarshal(d.Config.Data, &cfg)
		entities := waitForEntities(t, client, pluginID, d.ID, 20*time.Second)
		totalEntities += len(entities)

		rows := make([]string, 0, len(entities))
		for _, e := range entities {
			domainCounts[e.Domain]++
			rows = append(rows, fmt.Sprintf("%s(%s)", e.ID, e.Domain))
		}
		sort.Strings(rows)
		t.Logf("device %s (%s) addr=%q key_set=%t entities=%d: %s", d.ID, bestName(d), cfg.Address, strings.TrimSpace(cfg.Key) != "", len(entities), strings.Join(rows, ", "))
	}

	domains := make([]string, 0, len(domainCounts))
	for d := range domainCounts {
		domains = append(domains, d)
	}
	sort.Strings(domains)
	parts := make([]string, 0, len(domains))
	for _, d := range domains {
		parts = append(parts, fmt.Sprintf("%s=%d", d, domainCounts[d]))
	}
	t.Logf("target summary: devices=%d entities=%d domains=[%s]", len(targets), totalEntities, strings.Join(parts, ", "))

	if totalEntities == 0 {
		t.Fatalf("target devices found but no entities returned")
	}
}

func waitForDevices(t *testing.T, client http.Client, pluginID string, timeout time.Duration) []types.Device {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last []types.Device
	for time.Now().Before(deadline) {
		resp, err := client.Get(testutil.APIBaseURL() + "/api/plugins/" + pluginID + "/devices")
		if err == nil && resp.StatusCode == http.StatusOK {
			var devices []types.Device
			if decErr := json.NewDecoder(resp.Body).Decode(&devices); decErr == nil {
				resp.Body.Close()
				last = devices
				if len(devices) > 0 {
					return devices
				}
			} else {
				resp.Body.Close()
			}
		} else if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	return last
}

func filterTargetDevices(devices []types.Device) []types.Device {
	out := make([]types.Device, 0)
	for _, d := range devices {
		name := strings.ToLower(bestName(d) + " " + d.ID + " " + d.SourceID)
		if strings.Contains(name, "ratgdo") ||
			strings.Contains(name, "alex switch") ||
			strings.Contains(name, "alex room switch") ||
			strings.Contains(name, "alex-room-switch") ||
			strings.Contains(name, "alex_switch") {
			out = append(out, d)
		}
	}
	return out
}

func bestName(d types.Device) string {
	if strings.TrimSpace(d.LocalName) != "" {
		return d.LocalName
	}
	if strings.TrimSpace(d.SourceName) != "" {
		return d.SourceName
	}
	return d.ID
}

func waitForEntities(t *testing.T, client http.Client, pluginID, deviceID string, timeout time.Duration) []types.Entity {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last []types.Entity
	for time.Now().Before(deadline) {
		last = listESPHomeEntities(t, client, pluginID, deviceID)
		if len(last) > 0 {
			return last
		}
		time.Sleep(500 * time.Millisecond)
	}
	return last
}

func listESPHomeEntities(t *testing.T, client http.Client, pluginID, deviceID string) []types.Entity {
	t.Helper()
	resp, err := client.Get(testutil.APIBaseURL() + "/api/plugins/" + pluginID + "/devices/" + deviceID + "/entities")
	if err != nil {
		t.Fatalf("list entities for %s failed: %v", deviceID, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list entities for %s status=%d", deviceID, resp.StatusCode)
	}
	var entities []types.Entity
	if err := json.NewDecoder(resp.Body).Decode(&entities); err != nil {
		t.Fatalf("decode entities for %s failed: %v", deviceID, err)
	}
	return entities
}
