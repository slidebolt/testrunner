package integration

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	requiredPlugins := []string{
		"gateway",
		"plugin-test-clean",
		"plugin-test-slow",
		"plugin-test-flaky",
	}

	for _, id := range requiredPlugins {
		if !waitForPlugin(id, 20*time.Second) {
			fmt.Printf("required plugin %q did not become healthy within timeout\n", id)
			os.Exit(1)
		}
	}

	os.Exit(m.Run())
}
