resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Setting the nixpacks.toml path for an app
resource "dokku_builder_nixpacks_property" "example" {
  app      = dokku_app.node_js_app.app
  property = "nixpackstoml-path"
  value    = "config/nixpacks.toml"
}

# Setting the nixpacks.toml path globally
resource "dokku_builder_nixpacks_property" "example_2" {
  global   = true
  property = "nixpackstoml-path"
  value    = "nixpacks.toml"
}
