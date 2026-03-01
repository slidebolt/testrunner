package integration

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/slidebolt/testrunner/integration/testutil"
)

func TestMain(m *testing.M) {
	if !testutil.WaitForPlugin("gateway", 2*time.Second) {
		fmt.Printf("required plugin %q did not become healthy within timeout\n", "gateway")
		os.Exit(1)
	}

	os.Exit(m.Run())
}
