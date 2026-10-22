resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Setting the restart-policy value for an app
resource "dokku_ps_property" "example" {
  app      = dokku_app.node_js_app.app
  property = "restart-policy"
  value    = "on-failure:5"
}

# Setting the restart-policy value globally
resource "dokku_ps_property" "example_2" {
  global   = true
  property = "restart-policy"
  value    = "on-failure:5"
}
