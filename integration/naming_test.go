package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/slidebolt/sdk-types"
	"github.com/slidebolt/testrunner/integration/testutil"
)

// readDeviceFile reads and unmarshals a device JSON file from the plugin data dir.
func readDeviceFile(t *testing.T, dataDir, deviceID string) types.Device {
	t.Helper()
	path := filepath.Join(dataDir, "devices", deviceID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("device file missing at %s: %v", path, err)
	}
	var dev types.Device
	if err := json.Unmarshal(data, &dev); err != nil {
		t.Fatalf("device file invalid JSON: %v", err)
	}
	return dev
}

// readEntityFile reads and unmarshals an entity JSON file from the plugin data dir.
func readEntityFile(t *testing.T, dataDir, deviceID, entityID string) types.Entity {
	t.Helper()
	path := filepath.Join(dataDir, "devices", deviceID, "entities", entityID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("entity file missing at %s: %v", path, err)
	}
	var ent types.Entity
	if err := json.Unmarshal(data, &ent); err != nil {
		t.Fatalf("entity file invalid JSON: %v", err)
	}
	return ent
}

func putDevice(t *testing.T, client *http.Client, base string, dev types.Device) {
	t.Helper()
	body, _ := json.Marshal(dev)
	req, _ := http.NewRequest(http.MethodPut, base+"/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("PUT device: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT device: expected 200, got %d", resp.StatusCode)
	}
}

func putEntity(t *testing.T, client *http.Client, base, deviceID string, ent types.Entity) {
	t.Helper()
	body, _ := json.Marshal(ent)
	req, _ := http.NewRequest(http.MethodPut, base+"/devices/"+deviceID+"/entities", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("PUT entity: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT entity: expected 200, got %d", resp.StatusCode)
	}
}

func TestNameWalledGarden(t *testing.T) {
	const pluginID = "plugin-test-clean"
	testutil.RequirePlugin(t, pluginID)

	dataDir := testutil.PluginDataDir(pluginID)
	if dataDir == "" {
		t.Fatal("could not locate plugin data directory")
	}

	client := &http.Client{}
	base := testutil.APIBaseURL() + "/api/plugins/" + pluginID

	// ── Devices ──────────────────────────────────────────────────────────────

	t.Run("device: source update does not overwrite local_name", func(t *testing.T) {
		const id = "wg-device-source-update"

		// Create with source fields only.
		body, _ := json.Marshal(types.Device{ID: id, SourceID: "src-001", SourceName: "Source Name"})
		resp, _ := client.Post(base+"/devices", "application/json", bytes.NewReader(body))
		resp.Body.Close()

		// User sets local_name.
		putDevice(t, client, base, types.Device{ID: id, LocalName: "User Name"})

		// Source pushes an update — omits local_name entirely.
		putDevice(t, client, base, types.Device{ID: id, SourceID: "src-001", SourceName: "Source Name Updated"})

		dev := readDeviceFile(t, dataDir, id)
		if dev.LocalName != "User Name" {
			t.Errorf("local_name: got %q, want %q", dev.LocalName, "User Name")
		}
		if dev.SourceName != "Source Name Updated" {
			t.Errorf("source_name: got %q, want %q", dev.SourceName, "Source Name Updated")
		}
		fmt.Println("PASS: device local_name survived source update")
	})

	t.Run("device: setting source_id does not overwrite local_name", func(t *testing.T) {
		const id = "wg-device-source-id"

		body, _ := json.Marshal(types.Device{ID: id, LocalName: "User Name"})
		resp, _ := client.Post(base+"/devices", "application/json", bytes.NewReader(body))
		resp.Body.Close()

		// Update only source_id — local_name must survive.
		putDevice(t, client, base, types.Device{ID: id, SourceID: "new-src-id"})

		dev := readDeviceFile(t, dataDir, id)
		if dev.LocalName != "User Name" {
			t.Errorf("local_name: got %q, want %q", dev.LocalName, "User Name")
		}
		if dev.SourceID != "new-src-id" {
			t.Errorf("source_id: got %q, want %q", dev.SourceID, "new-src-id")
		}
		fmt.Println("PASS: device local_name survived source_id update")
	})

	t.Run("device: setting local_name does not overwrite source fields", func(t *testing.T) {
		const id = "wg-device-local-name"

		body, _ := json.Marshal(types.Device{ID: id, SourceID: "src-abc", SourceName: "Source Name"})
		resp, _ := client.Post(base+"/devices", "application/json", bytes.NewReader(body))
		resp.Body.Close()

		// User sets local_name — omits source fields.
		putDevice(t, client, base, types.Device{ID: id, LocalName: "User Name"})

		dev := readDeviceFile(t, dataDir, id)
		if dev.SourceID != "src-abc" {
			t.Errorf("source_id: got %q, want %q", dev.SourceID, "src-abc")
		}
		if dev.SourceName != "Source Name" {
			t.Errorf("source_name: got %q, want %q", dev.SourceName, "Source Name")
		}
		if dev.LocalName != "User Name" {
			t.Errorf("local_name: got %q, want %q", dev.LocalName, "User Name")
		}
		fmt.Println("PASS: device source fields survived local_name update")
	})

	// ── Entities ─────────────────────────────────────────────────────────────

	t.Run("entity: source update does not overwrite local_name", func(t *testing.T) {
		const deviceID = "wg-ent-device"
		const entityID = "wg-entity-source-update"

		// Ensure device exists.
		body, _ := json.Marshal(types.Device{ID: deviceID})
		resp, _ := client.Post(base+"/devices", "application/json", bytes.NewReader(body))
		resp.Body.Close()

		// Create entity with source fields.
		body, _ = json.Marshal(types.Entity{ID: entityID, Domain: "switch", Actions: []string{"turn_on", "turn_off"}})
		resp, _ = client.Post(base+"/devices/"+deviceID+"/entities", "application/json", bytes.NewReader(body))
		resp.Body.Close()

		// User sets local_name.
		putEntity(t, client, base, deviceID, types.Entity{ID: entityID, Domain: "switch", LocalName: "User Entity Name"})

		// Source pushes new actions — omits local_name.
		putEntity(t, client, base, deviceID, types.Entity{ID: entityID, Domain: "switch", Actions: []string{"turn_on", "turn_off", "toggle"}})

		ent := readEntityFile(t, dataDir, deviceID, entityID)
		if ent.LocalName != "User Entity Name" {
			t.Errorf("local_name: got %q, want %q", ent.LocalName, "User Entity Name")
		}
		if len(ent.Actions) != 3 {
			t.Errorf("actions: got %v, want 3 actions", ent.Actions)
		}
		fmt.Println("PASS: entity local_name survived source update")
	})

	t.Run("entity: setting local_name does not overwrite domain or actions", func(t *testing.T) {
		const deviceID = "wg-ent-device"
		const entityID = "wg-entity-local-name"

		body, _ := json.Marshal(types.Entity{ID: entityID, Domain: "switch", Actions: []string{"turn_on", "turn_off"}})
		resp, _ := client.Post(base+"/devices/"+deviceID+"/entities", "application/json", bytes.NewReader(body))
		resp.Body.Close()

		// User sets local_name only — omits domain and actions.
		putEntity(t, client, base, deviceID, types.Entity{ID: entityID, LocalName: "User Entity Name"})

		ent := readEntityFile(t, dataDir, deviceID, entityID)
		if ent.Domain != "switch" {
			t.Errorf("domain: got %q, want %q", ent.Domain, "switch")
		}
		if len(ent.Actions) != 2 {
			t.Errorf("actions: got %v, want 2 actions", ent.Actions)
		}
		if ent.LocalName != "User Entity Name" {
			t.Errorf("local_name: got %q, want %q", ent.LocalName, "User Entity Name")
		}
		fmt.Println("PASS: entity domain and actions survived local_name update")
	})
}
