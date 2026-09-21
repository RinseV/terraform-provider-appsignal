// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNotifiersDataSource(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing without a name filter.
			{
				Config: providerConfig + `data "appsignal_notifiers" "test" { app_id = "6aad24d1ba6bc351255e7cb5" }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.appsignal_notifiers.test", "app_id", "6aad24d1ba6bc351255e7cb5"),
					resource.TestCheckResourceAttrSet("data.appsignal_notifiers.test", "notifiers.#"),
				),
			},
			// A name that matches nothing returns an empty list rather than an error.
			{
				Config: providerConfig + `data "appsignal_notifiers" "test" {
  app_id = "6aad24d1ba6bc351255e7cb5"
  name   = "no-notifier-has-this-name"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.appsignal_notifiers.test", "notifiers.#", "0"),
				),
			},
		},
	})
}
