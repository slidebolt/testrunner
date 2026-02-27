package integration

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/slidebolt/testrunner/integration/testutil"
)

func TestRobustness(t *testing.T) {
	client := http.Client{}
	pid := "ghost-plugin"

	t.Run("Non-Existent Plugin Returns 403", func(t *testing.T) {
		resp, err := client.Get(testutil.APIBaseURL() + "/api/plugins/ghost-plugin/devices")
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
		url := fmt.Sprintf("%s/api/plugins/%s/devices", testutil.APIBaseURL(), pid)
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
}
