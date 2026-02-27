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

func TestSystemModelAndTickEvents(t *testing.T) {
	testutil.RequirePlugin(t, "plugin-system")
	client := http.Client{Timeout: 3 * time.Second}

	devices := listPluginDevices(t, client, "plugin-system")
	found := false
	for _, d := range devices {
		if d.ID == "system-device" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("system fixed device missing; got %v", devices)
	}

	entities := getEntitiesForDevice(t, client, "plugin-system", "system-device")
	required := map[string]bool{"system-time": false, "system-date": false, "system-cpu": false}
	for _, e := range entities {
		if _, ok := required[e.ID]; ok {
			required[e.ID] = true
		}
	}
	for id, ok := range required {
		if !ok {
			t.Fatalf("system entity %s missing; got %+v", id, entities)
		}
	}

	assertTickRate(t, client, "system-time")
	assertTickRate(t, client, "system-date")
	assertTickRate(t, client, "system-cpu")
}

type observedJournalEvent struct {
	Name      string    `json:"name"`
	PluginID  string    `json:"plugin_id"`
	DeviceID  string    `json:"device_id"`
	EntityID  string    `json:"entity_id"`
	EventID   string    `json:"event_id"`
	CreatedAt time.Time `json:"created_at"`
}

func assertTickRate(t *testing.T, client http.Client, entityID string) {
	t.Helper()
	first := waitForLatestSystemEvent(t, client, entityID, 8*time.Second)
	time.Sleep(1200 * time.Millisecond)
	second := waitForLatestSystemEvent(t, client, entityID, 8*time.Second)
	if first.EventID == second.EventID {
		t.Fatalf("expected new event id for %s after 1.2s, still %s", entityID, first.EventID)
	}
}

func waitForLatestSystemEvent(t *testing.T, client http.Client, entityID string, timeout time.Duration) observedJournalEvent {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		url := fmt.Sprintf("%s/api/journal/events?plugin_id=plugin-system&device_id=system-device&entity_id=%s", testutil.APIBaseURL(), entityID)
		resp, err := client.Get(url)
		if err != nil {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			time.Sleep(200 * time.Millisecond)
			continue
		}
		var events []observedJournalEvent
		if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
			resp.Body.Close()
			time.Sleep(200 * time.Millisecond)
			continue
		}
		resp.Body.Close()
		if len(events) > 0 {
			return events[len(events)-1]
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("no journal events for system entity %s within %s", entityID, timeout)
	return observedJournalEvent{}
}

func listPluginDevices(t *testing.T, client http.Client, pluginID string) []types.Device {
	t.Helper()
	url := fmt.Sprintf("%s/api/plugins/%s/devices", testutil.APIBaseURL(), pluginID)
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("failed to fetch devices for %s: %v", pluginID, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to fetch devices for %s, status=%d", pluginID, resp.StatusCode)
	}
	var devices []types.Device
	if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
		t.Fatalf("failed to decode devices: %v", err)
	}
	return devices
}

func getEntitiesForDevice(t *testing.T, client http.Client, pluginID, deviceID string) []types.Entity {
	t.Helper()
	url := fmt.Sprintf("%s/api/plugins/%s/devices/%s/entities", testutil.APIBaseURL(), pluginID, deviceID)
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("failed to fetch entities for %s/%s: %v", pluginID, deviceID, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to fetch entities for %s/%s, status=%d", pluginID, deviceID, resp.StatusCode)
	}
	var entities []types.Entity
	if err := json.NewDecoder(resp.Body).Decode(&entities); err != nil {
		t.Fatalf("failed to decode entities: %v", err)
	}
	return entities
}
