resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Enabling internal TLS for an app
resource "dokku_caddy_property" "example" {
  app      = dokku_app.node_js_app.app
  property = "tls-internal"
  value    = "true"
}

# Setting the letsencrypt email globally
resource "dokku_caddy_property" "example_2" {
  global   = true
  property = "letsencrypt-email"
  value    = "admin@example.com"
}
