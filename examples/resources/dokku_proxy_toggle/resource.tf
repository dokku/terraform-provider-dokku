resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Enable the proxy for an app
resource "dokku_proxy_toggle" "example" {
  app = dokku_app.node_js_app.app
}
