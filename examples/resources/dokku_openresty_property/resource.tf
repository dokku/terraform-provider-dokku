resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Setting the proxy read timeout for an app
resource "dokku_openresty_property" "example" {
  app      = dokku_app.node_js_app.app
  property = "proxy-read-timeout"
  value    = "120s"
}

# Setting the client max body size for an app
resource "dokku_openresty_property" "example_2" {
  app      = dokku_app.node_js_app.app
  property = "client-max-body-size"
  value    = "50m"
}

# Setting a global openresty property
resource "dokku_openresty_property" "example_3" {
  global   = true
  property = "bind-address-ipv4"
  value    = "0.0.0.0"
}
