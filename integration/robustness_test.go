package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/slidebolt/sdk-types"
)

func TestRobustness(t *testing.T) {
	client := http.Client{}
	pid := "plugin-test-clean"

	t.Run("Non-Existent Plugin Returns 403", func(t *testing.T) {
		resp, err := client.Get(apiBaseURL() + "/api/plugins/ghost-plugin/devices")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("Expected 403 for unregistered plugin, got %d", resp.StatusCode)
		}
		fmt.Println("PASS: Unregistered plugin access returns 403.")
	})

	t.Run("Malformed JSON Returns 400", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/plugins/%s/devices", apiBaseURL(), pid)
		resp, err := client.Post(url, "application/json", bytes.NewBuffer([]byte(`{invalid-json}`)))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected 400 for malformed JSON, got %d", resp.StatusCode)
		}
		fmt.Println("PASS: Malformed JSON correctly rejected with 400.")
	})

	t.Run("Device Update Round-Trip", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/plugins/%s/devices", apiBaseURL(), pid)

		dev := types.Device{ID: "update-me", LocalName: "Original"}
		payload, _ := json.Marshal(dev)
		client.Post(url, "application/json", bytes.NewBuffer(payload))

		dev.LocalName = "Modified"
		payload, _ = json.Marshal(dev)
		req, _ := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("PUT failed: err=%v status=%v", err, statusCode(resp))
		}
		resp.Body.Close()

		vResp, _ := client.Get(url)
		var devices []types.Device
		json.NewDecoder(vResp.Body).Decode(&devices)
		vResp.Body.Close()

		found := false
		for _, d := range devices {
			if d.ID == "update-me" && d.LocalName == "Modified" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Update not reflected: could not find device with LocalName=Modified")
		}
		fmt.Println("PASS: Device update round-trip verified.")
	})
}
