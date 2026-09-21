# Get a log source of an app by name
data "appsignal_log_source" "example" {
  app_id = "abcdef"
  name   = "application"
}
