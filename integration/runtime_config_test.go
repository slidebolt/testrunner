package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type testRuntimeConfig struct {
	APIBaseURL string `json:"api_base_url"`
}

var (
	runtimeOnce sync.Once
	runtimeCfg  testRuntimeConfig
)

func loadRuntimeConfig() {
	runtimeCfg = testRuntimeConfig{APIBaseURL: "http://127.0.0.1:8082"}
	if v := os.Getenv("TEST_API_BASE_URL"); v != "" {
		runtimeCfg.APIBaseURL = v
		return
	}
	wd, err := os.Getwd()
	if err != nil {
		return
	}
	// tests run from testrunner/integration/ — go up two levels to test root
	path := filepath.Join(wd, "..", "..", ".build", "runtime.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &runtimeCfg)
	if runtimeCfg.APIBaseURL == "" {
		runtimeCfg.APIBaseURL = "http://127.0.0.1:8082"
	}
}

func apiBaseURL() string {
	runtimeOnce.Do(loadRuntimeConfig)
	return runtimeCfg.APIBaseURL
}
