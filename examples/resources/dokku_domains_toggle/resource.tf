resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Enable the domains plugin for an app
resource "dokku_domains_toggle" "example" {
  app = dokku_app.node_js_app.app
}
