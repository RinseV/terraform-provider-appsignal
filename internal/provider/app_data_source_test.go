// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAppDataSource(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing.
			{
				Config: providerConfig + `data "appsignal_app" "test" { id = "69808b0ce5250a3229a9f634" }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.appsignal_app.test", "id"),
					resource.TestCheckResourceAttrSet("data.appsignal_app.test", "name"),
					resource.TestCheckResourceAttrSet("data.appsignal_app.test", "environment"),
					resource.TestCheckResourceAttrSet("data.appsignal_app.test", "created_at"),
					resource.TestCheckResourceAttrSet("data.appsignal_app.test", "updated_at"),
				),
			},
		},
	})
}
