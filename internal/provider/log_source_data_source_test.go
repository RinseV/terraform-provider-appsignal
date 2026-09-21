package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// testAccLogSourceDataSourceConfig creates an app with a log source so the
// data source has something of its own to read back.
func testAccLogSourceDataSourceConfig(name string) string {
	return providerConfig + fmt.Sprintf(`
resource "appsignal_app" "test" {
  name        = "my-test-log-source-data-source-app"
  environment = "testing"
}

resource "appsignal_log_source" "test" {
  app_id = appsignal_app.test.id
  name   = "my-test-log-source"
  type   = "http"
  fmt    = "JSON"
}

data "appsignal_log_source" "test" {
  app_id = appsignal_app.test.id
  name   = %[1]q

  depends_on = [appsignal_log_source.test]
}
`, name)
}

func TestAccLogSourceDataSource(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing.
			{
				Config: testAccLogSourceDataSourceConfig("my-test-log-source"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.appsignal_log_source.test", "app_id", "appsignal_app.test", "id"),
					resource.TestCheckResourceAttrPair("data.appsignal_log_source.test", "id", "appsignal_log_source.test", "id"),
					resource.TestCheckResourceAttr("data.appsignal_log_source.test", "name", "my-test-log-source"),
					resource.TestCheckResourceAttr("data.appsignal_log_source.test", "type", "http"),
					resource.TestCheckResourceAttr("data.appsignal_log_source.test", "fmt", "JSON"),
					resource.TestCheckResourceAttrSet("data.appsignal_log_source.test", "key"),
				),
			},
			// A name that matches no log source is an error.
			{
				Config:      testAccLogSourceDataSourceConfig("no-log-source-has-this-name"),
				ExpectError: regexp.MustCompile(`AppSignal Log Source Not Found`),
			},
		},
	})
}
