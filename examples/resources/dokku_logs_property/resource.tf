resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Setting the max-size value for an app
resource "dokku_logs_property" "example" {
  app      = dokku_app.node_js_app.app
  property = "max-size"
  value    = "100m"
}

# Setting the max-size value globally
resource "dokku_logs_property" "example_2" {
  global   = true
  property = "max-size"
  value    = "100m"
}
