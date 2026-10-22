resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Setting the railpack.json path for an app
resource "dokku_builder_railpack_property" "example" {
  app      = dokku_app.node_js_app.app
  property = "railpackjson-path"
  value    = "config/railpack.json"
}

# Setting the railpack.json path globally
resource "dokku_builder_railpack_property" "example_2" {
  global   = true
  property = "railpackjson-path"
  value    = "railpack.json"
}
