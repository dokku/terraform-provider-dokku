resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Setting the appjson-path for an app
resource "dokku_app_json_property" "example" {
  app      = dokku_app.node_js_app.app
  property = "appjson-path"
  value    = "app.json"
}

# Setting the appjson-path globally
resource "dokku_app_json_property" "example_2" {
  global   = true
  property = "appjson-path"
  value    = "app.json"
}
