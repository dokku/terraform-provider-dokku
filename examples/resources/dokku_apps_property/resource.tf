# Disabling app auto-creation globally
resource "dokku_apps_property" "example" {
  global   = true
  property = "disable-autocreation"
  value    = "true"
}

resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Overriding the deploy-source for an app
resource "dokku_apps_property" "example_2" {
  app      = dokku_app.node_js_app.app
  property = "deploy-source"
  value    = "git"
}

# Overriding the deploy-source-metadata for an app
resource "dokku_apps_property" "example_3" {
  app      = dokku_app.node_js_app.app
  property = "deploy-source-metadata"
  value    = "https://example.com/repo"
}
