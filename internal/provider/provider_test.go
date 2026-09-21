package provider

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// providerConfig is a shared configuration to combine with the actual test
// configuration so the AppSignal client is properly configured. The token is
// deliberately left out: the provider reads APPSIGNAL_TOKEN from the
// environment, which keeps it out of the test source and out of any Terraform
// state the test harness writes. The host is left out as well, so the tests
// run against the default endpoint unless APPSIGNAL_HOST points them
// elsewhere.
const providerConfig = `
provider "appsignal" {
  organization = "terraform-test"
}
`

var (
	// testAccProtoV6ProviderFactories are used to instantiate a provider during
	// acceptance testing. The factory function will be invoked for every Terraform
	// CLI command executed to create a provider server to which the CLI can
	// reattach.
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"appsignal": providerserver.NewProtocol6WithError(New("test")()),
	}
)

func TestMain(m *testing.M) {
	loadEnvFile(".env")
	os.Exit(m.Run())
}

// testAccPreCheck skips an acceptance test when no token is available, so a run
// without credentials reports a skip rather than a wall of API errors.
func testAccPreCheck(t *testing.T) {
	t.Helper()

	if os.Getenv("APPSIGNAL_TOKEN") == "" {
		t.Skip("APPSIGNAL_TOKEN not set, skipping acceptance test")
	}
}

// loadEnvFile reads a .env file into the process environment so the tests can
// be run without any shell or IDE setup. Go runs the test binary from the
// package directory, so name is looked up there and in every parent directory
// up to the filesystem root, which is what finds the file in the repository
// root. Variables already present in the environment are left alone, so the
// shell and CI keep the final say. A missing or unreadable file is ignored:
// the tests skip on a missing token anyway.
func loadEnvFile(name string) {
	path, err := findUp(name)
	if err != nil {
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		value = strings.TrimSpace(value)
		if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
			value = value[1 : len(value)-1]
		}

		if _, set := os.LookupEnv(key); set {
			continue
		}
		os.Setenv(key, value)
	}
}

// findUp returns the path to name in the working directory or the nearest
// parent directory that holds it.
func findUp(name string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fs.ErrNotExist
		}
		dir = parent
	}
}
