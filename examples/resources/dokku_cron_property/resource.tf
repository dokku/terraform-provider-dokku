resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Enabling maintenance mode for an app
resource "dokku_cron_property" "example" {
  app      = dokku_app.node_js_app.app
  property = "maintenance"
  value    = "true"
}

# Setting the mailto address globally
resource "dokku_cron_property" "example_2" {
  global   = true
  property = "mailto"
  value    = "ops@example.com"
}
