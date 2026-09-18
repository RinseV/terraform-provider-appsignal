// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccLogTriggerConfig returns a configuration with an app and a log source
// to hang the log trigger off, so the test owns every resource it touches and
// cleans them all up again. description is a complete HCL attribute line so a
// test step can leave it out entirely, which is how a cleared description is
// expressed.
func testAccLogTriggerConfig(name, query, severities, description string) string {
	return providerConfig + fmt.Sprintf(`
resource "appsignal_app" "test" {
  name        = "my-test-log-trigger-app"
  environment = "testing"
}

resource "appsignal_log_source" "test" {
  app_id = appsignal_app.test.id
  name   = "my-test-log-trigger-log-source"
  type   = "http"
  fmt    = "PLAINTEXT"
}

resource "appsignal_log_trigger" "test" {
  app_id     = appsignal_app.test.id
  source_ids = [appsignal_log_source.test.id]
  name       = %[1]q
  query      = %[2]q
  severities = %[3]s
  %[4]s
}
`, name, query, severities, description)
}

func TestAccLogTriggerResource(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// A severity outside the enum is rejected before anything is
			// created, rather than by the API.
			{
				Config: testAccLogTriggerConfig(
					"my-test-log-trigger",
					"hostname=web",
					`["fatal"]`,
					``,
				),
				ExpectError: regexp.MustCompile(`Attribute severities\[Value\("fatal"\)\] value must be one of`),
			},
			// Create and Read testing
			{
				Config: testAccLogTriggerConfig(
					"my-test-log-trigger",
					"hostname=web",
					`["FATAL"]`,
					`description = "created by the acceptance tests"`,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appsignal_log_trigger.test", "name", "my-test-log-trigger"),
					resource.TestCheckResourceAttr("appsignal_log_trigger.test", "query", "hostname=web"),
					resource.TestCheckResourceAttr("appsignal_log_trigger.test", "description", "created by the acceptance tests"),
					resource.TestCheckResourceAttr("appsignal_log_trigger.test", "severities.#", "1"),
					resource.TestCheckTypeSetElemAttr("appsignal_log_trigger.test", "severities.*", "FATAL"),
					resource.TestCheckResourceAttr("appsignal_log_trigger.test", "source_ids.#", "1"),
					// The defaults the schema fills in when the configuration
					// stays quiet about them.
					resource.TestCheckResourceAttr("appsignal_log_trigger.test", "notification_options", "ALWAYS"),
					resource.TestCheckResourceAttr("appsignal_log_trigger.test", "notification_trigger_value", "1"),
					resource.TestCheckResourceAttr("appsignal_log_trigger.test", "notifier_ids.#", "0"),
					resource.TestCheckResourceAttr("appsignal_log_trigger.test", "notifiers.#", "0"),
					resource.TestCheckResourceAttrSet("appsignal_log_trigger.test", "id"),
					resource.TestCheckResourceAttrSet("appsignal_log_trigger.test", "action_type"),
					resource.TestCheckResourceAttrSet("appsignal_log_trigger.test", "order"),
					resource.TestCheckResourceAttrPair(
						"appsignal_log_trigger.test", "app_id",
						"appsignal_app.test", "id",
					),
					resource.TestCheckTypeSetElemAttrPair(
						"appsignal_log_trigger.test", "source_ids.*",
						"appsignal_log_source.test", "id",
					),
				),
			},
			// Update and Read testing
			{
				Config: testAccLogTriggerConfig(
					"my-test-log-trigger-updated",
					"hostname=worker",
					`["FATAL", "WARN"]`,
					`description = "updated by the acceptance tests"`,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appsignal_log_trigger.test", "name", "my-test-log-trigger-updated"),
					resource.TestCheckResourceAttr("appsignal_log_trigger.test", "query", "hostname=worker"),
					resource.TestCheckResourceAttr("appsignal_log_trigger.test", "description", "updated by the acceptance tests"),
					resource.TestCheckResourceAttr("appsignal_log_trigger.test", "severities.#", "2"),
					resource.TestCheckTypeSetElemAttr("appsignal_log_trigger.test", "severities.*", "FATAL"),
					resource.TestCheckTypeSetElemAttr("appsignal_log_trigger.test", "severities.*", "WARN"),
					resource.TestCheckResourceAttr("appsignal_log_trigger.test", "source_ids.#", "1"),
				),
			},
			// Clearing the description, which the API only accepts as an empty
			// string and which has to read back as a null attribute again.
			{
				Config: testAccLogTriggerConfig(
					"my-test-log-trigger-updated",
					"hostname=worker",
					`["FATAL", "WARN"]`,
					``,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("appsignal_log_trigger.test", "description"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "appsignal_log_trigger.test",
				ImportState:       true,
				ImportStateVerify: true,
				// A log trigger is only readable through its app, so the import
				// ID carries both IDs.
				ImportStateIdFunc: testAccLogTriggerImportStateID("appsignal_log_trigger.test"),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// testAccLogTriggerImportStateID builds the "<app id>,<log trigger id>" import
// ID from the state the earlier steps left behind.
func testAccLogTriggerImportStateID(resourceName string) resource.ImportStateIdFunc {
	return func(state *terraform.State) (string, error) {
		rs, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", resourceName)
		}

		return fmt.Sprintf("%s,%s", rs.Primary.Attributes["app_id"], rs.Primary.ID), nil
	}
}
