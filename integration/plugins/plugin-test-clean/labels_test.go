package plugintestclean

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/slidebolt/sdk-types"
	"github.com/slidebolt/testrunner/integration/testutil"
)

func TestLabelSearch(t *testing.T) {
	testutil.RequirePlugin(t, pluginID)

	client := http.Client{Timeout: 2 * time.Second}
	deviceID := "label-dev-1"
	entityID := "label-entity-1"
	devURL := fmt.Sprintf("%s/api/plugins/%s/devices", testutil.APIBaseURL(), pluginID)
	entURL := fmt.Sprintf("%s/api/plugins/%s/devices/%s/entities", testutil.APIBaseURL(), pluginID, deviceID)

	// --- setup: create device and entity with labels ---

	dev := types.Device{
		ID:        deviceID,
		SourceID:  "src-label-1",
		LocalName: "Label Test Device",
		Labels: map[string]string{
			"room":  "living-room",
			"floor": "ground",
		},
	}
	payload, _ := json.Marshal(dev)
	resp, err := client.Post(devURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		t.Fatalf("create device failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create device unexpected status: %d", resp.StatusCode)
	}

	ent := types.Entity{
		ID:       entityID,
		DeviceID: deviceID,
		Domain:   "switch",
		Labels: map[string]string{
			"group": "lights",
		},
	}
	payload, _ = json.Marshal(ent)
	resp, err = client.Post(entURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		t.Fatalf("create entity failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create entity unexpected status: %d", resp.StatusCode)
	}

	// --- device label search ---

	t.Run("Device: single label match", func(t *testing.T) {
		results := searchDevices(t, client, "label=room:living-room")
		if !containsDevice(results, deviceID) {
			t.Fatalf("expected device %q in results, got %d result(s)", deviceID, len(results))
		}
		fmt.Printf("PASS: found device by label room:living-room\n")
	})

	t.Run("Device: two labels both match (AND)", func(t *testing.T) {
		results := searchDevices(t, client, "label=room:living-room&label=floor:ground")
		if !containsDevice(results, deviceID) {
			t.Fatalf("expected device %q in results, got %d result(s)", deviceID, len(results))
		}
		fmt.Printf("PASS: found device by AND labels room:living-room + floor:ground\n")
	})

	t.Run("Device: two labels one mismatch (AND)", func(t *testing.T) {
		results := searchDevices(t, client, "label=room:living-room&label=floor:upstairs")
		if containsDevice(results, deviceID) {
			t.Fatalf("expected device %q to be excluded, but it was returned", deviceID)
		}
		fmt.Printf("PASS: device correctly excluded when one label does not match\n")
	})

	t.Run("Device: label with no match", func(t *testing.T) {
		results := searchDevices(t, client, "label=room:kitchen")
		if containsDevice(results, deviceID) {
			t.Fatalf("expected device %q to be excluded, but it was returned", deviceID)
		}
		fmt.Printf("PASS: device correctly excluded for non-matching label\n")
	})

	// --- entity label search ---

	t.Run("Entity: single label match", func(t *testing.T) {
		results := searchEntities(t, client, "label=group:lights")
		if !containsEntity(results, entityID) {
			t.Fatalf("expected entity %q in results, got %d result(s)", entityID, len(results))
		}
		fmt.Printf("PASS: found entity by label group:lights\n")
	})

	t.Run("Entity: label with no match", func(t *testing.T) {
		results := searchEntities(t, client, "label=group:sensors")
		if containsEntity(results, entityID) {
			t.Fatalf("expected entity %q to be excluded, but it was returned", entityID)
		}
		fmt.Printf("PASS: entity correctly excluded for non-matching label\n")
	})
}

func searchDevices(t *testing.T, client http.Client, query string) []types.Device {
	t.Helper()
	resp, err := client.Get(fmt.Sprintf("%s/api/search/devices?q=*&%s", testutil.APIBaseURL(), query))
	if err != nil {
		t.Fatalf("device search request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("device search unexpected status: %d", resp.StatusCode)
	}
	var results []types.Device
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		t.Fatalf("decode device search results failed: %v", err)
	}
	return results
}

func searchEntities(t *testing.T, client http.Client, query string) []types.Entity {
	t.Helper()
	resp, err := client.Get(fmt.Sprintf("%s/api/search/entities?%s", testutil.APIBaseURL(), query))
	if err != nil {
		t.Fatalf("entity search request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("entity search unexpected status: %d", resp.StatusCode)
	}
	var results []types.Entity
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		t.Fatalf("decode entity search results failed: %v", err)
	}
	return results
}

func containsDevice(devices []types.Device, id string) bool {
	for _, d := range devices {
		if d.ID == id {
			return true
		}
	}
	return false
}

func containsEntity(entities []types.Entity, id string) bool {
	for _, e := range entities {
		if e.ID == id {
			return true
		}
	}
	return false
}
