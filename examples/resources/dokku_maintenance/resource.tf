resource "dokku_plugin" "maintenance" {
  name = "maintenance"
  url  = "https://github.com/dokku/dokku-maintenance.git"
}

resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Enable maintenance mode for an app
resource "dokku_maintenance" "example" {
  app        = dokku_app.node_js_app.app
  depends_on = [dokku_plugin.maintenance]
}
