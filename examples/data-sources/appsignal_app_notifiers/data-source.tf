# Get all notifiers of an app
data "appsignal_app_notifiers" "all" {
  app_id = "abcdef"
}

# Get the notifiers of an app with a specific name
data "appsignal_app_notifiers" "slack" {
  app_id = "abcdef"
  name   = "Slack"
}
