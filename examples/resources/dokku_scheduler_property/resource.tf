resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Selecting the scheduler for an app
resource "dokku_scheduler_property" "example" {
  app      = dokku_app.node_js_app.app
  property = "selected"
  value    = "docker-local"
}

# Selecting the scheduler globally
resource "dokku_scheduler_property" "example_2" {
  global   = true
  property = "selected"
  value    = "docker-local"
}
