resource "dokku_plugin" "maintenance" {
  name = "maintenance"
  url  = "https://github.com/dokku/dokku-maintenance.git"
}

resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Set a custom maintenance page from inline HTML
resource "dokku_maintenance_custom_page" "example" {
  app        = dokku_app.node_js_app.app
  content    = "<html><body><h1>Down for maintenance</h1></body></html>\n"
  depends_on = [dokku_plugin.maintenance]
}
