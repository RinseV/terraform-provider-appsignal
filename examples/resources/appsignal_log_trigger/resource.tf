resource "appsignal_app" "example" {
  name        = "my-example-app"
  environment = "production"
}

resource "appsignal_log_source" "example" {
  app_id = appsignal_app.example.id

  name = "my-example-log-source"
  fmt  = "AUTODETECT"
  type = "application"
}

resource "appsignal_log_trigger" "example" {
  app_id = appsignal_app.example.id

  source_ids = [appsignal_log_source.example.id]

  name  = "my-trigger"
  query = "hostname=web"

  severities = ["FATAL"]
}