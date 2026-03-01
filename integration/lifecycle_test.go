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

func TestDeviceAndEntityLifecycle(t *testing.T) {
	const pluginID = "plugin-test-clean"
	const deviceID = "lifecycle-device-001"
	const entityID = "lifecycle-entity-001"
	testutil.RequirePlugin(t, pluginID)

	dataDir := testutil.PluginDataDir(pluginID)
	if dataDir == "" {
		t.Fatal("could not locate plugin data directory")
	}

	client := http.Client{Timeout: 2 * time.Second}
	base := testutil.APIBaseURL() + "/api/plugins/" + pluginID
	deviceFile := filepath.Join(dataDir, "devices", deviceID+".json")
	entityFile := filepath.Join(dataDir, "devices", deviceID, "entities", entityID+".json")

	// --- Device ---

	t.Run("device create writes file", func(t *testing.T) {
		body, _ := json.Marshal(types.Device{ID: deviceID, LocalName: "Lifecycle Device"})
		resp, err := client.Post(base+"/devices", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("create device: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		data, err := os.ReadFile(deviceFile)
		if err != nil {
			t.Fatalf("device file missing after create: %v", err)
		}
		var dev types.Device
		if err := json.Unmarshal(data, &dev); err != nil {
			t.Fatalf("device file invalid JSON: %v", err)
		}
		if dev.ID != deviceID {
			t.Errorf("got device ID %q, want %q", dev.ID, deviceID)
		}
		fmt.Printf("PASS: device file written at %s\n", deviceFile)
	})

	t.Run("device update is reflected in file", func(t *testing.T) {
		body, _ := json.Marshal(types.Device{ID: deviceID, LocalName: "Lifecycle Device Updated"})
		req, _ := http.NewRequest(http.MethodPut, base+"/devices", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("update device: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		data, err := os.ReadFile(deviceFile)
		if err != nil {
			t.Fatalf("device file missing after update: %v", err)
		}
		var dev types.Device
		if err := json.Unmarshal(data, &dev); err != nil {
			t.Fatalf("device file invalid JSON: %v", err)
		}
		if dev.LocalName != "Lifecycle Device Updated" {
			t.Errorf("got LocalName %q, want %q", dev.LocalName, "Lifecycle Device Updated")
		}
		fmt.Printf("PASS: device file reflects updated LocalName\n")
	})

	// --- Entity ---

	t.Run("entity create writes file", func(t *testing.T) {
		body, _ := json.Marshal(types.Entity{ID: entityID, Domain: "switch", LocalName: "Lifecycle Entity"})
		resp, err := client.Post(base+"/devices/"+deviceID+"/entities", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("create entity: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		data, err := os.ReadFile(entityFile)
		if err != nil {
			t.Fatalf("entity file missing after create: %v", err)
		}
		var ent types.Entity
		if err := json.Unmarshal(data, &ent); err != nil {
			t.Fatalf("entity file invalid JSON: %v", err)
		}
		if ent.ID != entityID {
			t.Errorf("got entity ID %q, want %q", ent.ID, entityID)
		}
		fmt.Printf("PASS: entity file written at %s\n", entityFile)
	})

	t.Run("entity update is reflected in file", func(t *testing.T) {
		body, _ := json.Marshal(types.Entity{ID: entityID, Domain: "switch", LocalName: "Lifecycle Entity Updated"})
		req, _ := http.NewRequest(http.MethodPut, base+"/devices/"+deviceID+"/entities", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("update entity: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		data, err := os.ReadFile(entityFile)
		if err != nil {
			t.Fatalf("entity file missing after update: %v", err)
		}
		var ent types.Entity
		if err := json.Unmarshal(data, &ent); err != nil {
			t.Fatalf("entity file invalid JSON: %v", err)
		}
		if ent.LocalName != "Lifecycle Entity Updated" {
			t.Errorf("got LocalName %q, want %q", ent.LocalName, "Lifecycle Entity Updated")
		}
		fmt.Printf("PASS: entity file reflects updated LocalName\n")
	})

	t.Run("device delete removes device and entity files", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, base+"/devices/"+deviceID, nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("delete device: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		if _, err := os.Stat(deviceFile); !os.IsNotExist(err) {
			t.Errorf("device file still exists after delete: %s", deviceFile)
		}
		if _, err := os.Stat(entityFile); !os.IsNotExist(err) {
			t.Errorf("entity file still exists after device delete: %s", entityFile)
		}
		fmt.Printf("PASS: device and entity files removed\n")
	})
}
