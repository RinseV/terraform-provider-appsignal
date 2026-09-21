# Get every app of the organization configured on the provider
data "appsignal_apps" "all" {}

# The apps are keyed by ID, so they can be iterated over directly
resource "appsignal_log_source" "default" {
  for_each = data.appsignal_apps.all.apps

  app_id = each.key

  name = "default"
  fmt  = "AUTODETECT"
  type = "application"
}
