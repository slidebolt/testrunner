package pluginautomation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/slidebolt/sdk-types"
	"github.com/slidebolt/testrunner/integration/testutil"
)

func TestLuaScriptCommandUpdatesLuaState(t *testing.T) {
	const pluginID = "plugin-automation"
	testutil.RequirePlugin(t, pluginID)

	client := http.Client{Timeout: 3 * time.Second}
	deviceID := "automation-script-device"
	entityID := "party-switch"

	createDevice(t, client, pluginID, deviceID)
	createEntity(t, client, pluginID, deviceID, entityID)

	scriptPath, statePath := scriptPaths(t, pluginID, deviceID, entityID)
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("mkdir script dir failed: %v", err)
	}
	script := fmt.Sprintf(`
function OnInit(Ctx)
  Ctx:OnCommand("%s.%s.%s.PowerOn", "DoPowerOn")
end

function DoPowerOn(Ctx, Command)
  local c = Ctx:GetState("press_count")
  if c == nil then c = 0 end
  Ctx:SetState("press_count", c + 1)
end
`, pluginID, deviceID, entityID)
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write script failed: %v", err)
	}
	_ = os.Remove(statePath)

	postCommand(t, client, pluginID, deviceID, entityID, map[string]any{"type": "PowerOn"})
	waitForScriptCount(t, statePath, "press_count", 1, 5*time.Second)
}

func createDevice(t *testing.T, client http.Client, pluginID, deviceID string) {
	t.Helper()
	url := fmt.Sprintf("%s/api/plugins/%s/devices", testutil.APIBaseURL(), pluginID)
	dev := types.Device{ID: deviceID, SourceID: "src-" + deviceID, SourceName: "Automation Script Device", LocalName: "Automation Script Device"}
	payload, _ := json.Marshal(dev)
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		t.Fatalf("create device failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create device unexpected status: %d", resp.StatusCode)
	}
}

func createEntity(t *testing.T, client http.Client, pluginID, deviceID, entityID string) {
	t.Helper()
	url := fmt.Sprintf("%s/api/plugins/%s/devices/%s/entities", testutil.APIBaseURL(), pluginID, deviceID)
	ent := types.Entity{ID: entityID, Domain: "switch", LocalName: "Party Switch"}
	payload, _ := json.Marshal(ent)
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		t.Fatalf("create entity failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create entity unexpected status: %d", resp.StatusCode)
	}
}

func postCommand(t *testing.T, client http.Client, pluginID, deviceID, entityID string, payload map[string]any) {
	t.Helper()
	url := fmt.Sprintf("%s/api/plugins/%s/devices/%s/entities/%s/commands", testutil.APIBaseURL(), pluginID, deviceID, entityID)
	body, _ := json.Marshal(payload)
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("post command failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("post command unexpected status: %d", resp.StatusCode)
	}
}

func scriptPaths(t *testing.T, pluginID, deviceID, entityID string) (string, string) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	path := wd
	for i := 0; i < 8; i++ {
		runtimePath := filepath.Join(path, ".build", "runtime.json")
		if _, err := os.Stat(runtimePath); err == nil {
			base := filepath.Join(path, ".build", "data", pluginID, "devices", deviceID, "entities")
			return filepath.Join(base, entityID+".lua"), filepath.Join(base, entityID+".state.lua.json")
		}
		altRuntimePath := filepath.Join(path, "test", ".build", "runtime.json")
		if _, err := os.Stat(altRuntimePath); err == nil {
			base := filepath.Join(path, "test", ".build", "data", pluginID, "devices", deviceID, "entities")
			return filepath.Join(base, entityID+".lua"), filepath.Join(base, entityID+".state.lua.json")
		}
		next := filepath.Dir(path)
		if next == path {
			break
		}
		path = next
	}
	t.Fatalf("could not locate .build/runtime.json from %s", wd)
	return "", ""
}

func waitForScriptCount(t *testing.T, statePath, key string, want int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(statePath)
		if err == nil {
			var state map[string]any
			if err := json.Unmarshal(data, &state); err == nil {
				if got, ok := state[key].(float64); ok && int(got) == want {
					return
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("script state %s did not reach %s=%d within %s", statePath, key, want, timeout)
}
