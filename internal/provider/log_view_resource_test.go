package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccLogViewConfig returns a configuration with an app and a log source to
// hang the log view off, so the test owns every resource it touches and cleans
// them all up again. query, columns and line_height are complete HCL attribute
// lines so a test step can leave them out entirely.
func testAccLogViewConfig(name, query, severities, columns, lineHeight string) string {
	return providerConfig + fmt.Sprintf(`
resource "appsignal_app" "test" {
  name        = "my-test-log-view-app"
  environment = "testing"
}

resource "appsignal_log_source" "test" {
  app_id = appsignal_app.test.id
  name   = "my-test-log-view-log-source"
  type   = "http"
  fmt    = "PLAINTEXT"
}

resource "appsignal_log_view" "test" {
  app_id     = appsignal_app.test.id
  source_ids = [appsignal_log_source.test.id]
  name       = %[1]q
  severities = %[3]s
  %[2]s
  %[4]s
  %[5]s
}
`, name, query, severities, columns, lineHeight)
}

func TestAccLogViewResource(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// A severity outside the enum is rejected before anything is
			// created, rather than by the API.
			{
				Config: testAccLogViewConfig(
					"my-test-log-view",
					`query = "hostname=web"`,
					`["fatal"]`,
					``,
					``,
				),
				ExpectError: regexp.MustCompile(`Attribute severities\[Value\("fatal"\)\] value must be one of`),
			},
			// Create and Read testing
			{
				Config: testAccLogViewConfig(
					"my-test-log-view",
					`query = "hostname=web"`,
					`["FATAL"]`,
					`columns = ["timestamp", "message"]`,
					``,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appsignal_log_view.test", "name", "my-test-log-view"),
					resource.TestCheckResourceAttr("appsignal_log_view.test", "query", "hostname=web"),
					// The default the schema fills in when the configuration
					// stays quiet about it, which is what AppSignal assigns.
					resource.TestCheckResourceAttr("appsignal_log_view.test", "line_height", "0"),
					resource.TestCheckResourceAttr("appsignal_log_view.test", "severities.#", "1"),
					resource.TestCheckTypeSetElemAttr("appsignal_log_view.test", "severities.*", "FATAL"),
					resource.TestCheckResourceAttr("appsignal_log_view.test", "columns.#", "2"),
					resource.TestCheckTypeSetElemAttr("appsignal_log_view.test", "columns.*", "timestamp"),
					resource.TestCheckTypeSetElemAttr("appsignal_log_view.test", "columns.*", "message"),
					resource.TestCheckResourceAttr("appsignal_log_view.test", "source_ids.#", "1"),
					resource.TestCheckResourceAttrSet("appsignal_log_view.test", "id"),
					resource.TestCheckResourceAttrPair(
						"appsignal_log_view.test", "app_id",
						"appsignal_app.test", "id",
					),
					resource.TestCheckTypeSetElemAttrPair(
						"appsignal_log_view.test", "source_ids.*",
						"appsignal_log_source.test", "id",
					),
				),
			},
			// Update and Read testing
			{
				Config: testAccLogViewConfig(
					"my-test-log-view-updated",
					`query = "hostname=worker"`,
					`["FATAL", "WARN"]`,
					`columns = ["timestamp", "severity", "message"]`,
					`line_height = "relaxed"`,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appsignal_log_view.test", "name", "my-test-log-view-updated"),
					resource.TestCheckResourceAttr("appsignal_log_view.test", "query", "hostname=worker"),
					resource.TestCheckResourceAttr("appsignal_log_view.test", "line_height", "relaxed"),
					resource.TestCheckResourceAttr("appsignal_log_view.test", "severities.#", "2"),
					resource.TestCheckTypeSetElemAttr("appsignal_log_view.test", "severities.*", "FATAL"),
					resource.TestCheckTypeSetElemAttr("appsignal_log_view.test", "severities.*", "WARN"),
					resource.TestCheckResourceAttr("appsignal_log_view.test", "columns.#", "3"),
					resource.TestCheckResourceAttr("appsignal_log_view.test", "source_ids.#", "1"),
				),
			},
			// Clearing the query and the columns, which the schema defaults
			// back to their zero values rather than leaving them at their
			// previous ones.
			{
				Config: testAccLogViewConfig(
					"my-test-log-view-updated",
					``,
					`["FATAL", "WARN"]`,
					``,
					`line_height = "relaxed"`,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appsignal_log_view.test", "query", ""),
					resource.TestCheckResourceAttr("appsignal_log_view.test", "columns.#", "0"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "appsignal_log_view.test",
				ImportState:       true,
				ImportStateVerify: true,
				// A log view is only readable through its app, so the import
				// ID carries both IDs.
				ImportStateIdFunc: testAccLogViewImportStateID("appsignal_log_view.test"),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// testAccLogViewImportStateID builds the "<app id>,<log view id>" import ID
// from the state the earlier steps left behind.
func testAccLogViewImportStateID(resourceName string) resource.ImportStateIdFunc {
	return func(state *terraform.State) (string, error) {
		rs, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", resourceName)
		}

		return fmt.Sprintf("%s,%s", rs.Primary.Attributes["app_id"], rs.Primary.ID), nil
	}
}
