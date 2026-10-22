resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Setting the proxy type for an app
resource "dokku_proxy_property" "example" {
  app      = dokku_app.node_js_app.app
  property = "type"
  value    = "nginx"
}

# Setting the proxy type globally
resource "dokku_proxy_property" "example_2" {
  global   = true
  property = "type"
  value    = "haproxy"
}

# Setting the proxy port for an app
resource "dokku_proxy_property" "example_3" {
  app      = dokku_app.node_js_app.app
  property = "proxy-port"
  value    = "8080"
}
