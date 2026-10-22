resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Add an SSL certificate to an app
resource "dokku_certs" "example" {
  app  = dokku_app.node_js_app.app
  cert = "/etc/nginx/ssl/node-js-app.crt"
  key  = "/etc/nginx/ssl/node-js-app.key"
}

resource "dokku_plugin" "global_cert" {
  name = "global-cert"
  url  = "https://github.com/dokku-community/dokku-global-cert.git"
}

# Add a global SSL certificate (requires the dokku-global-cert plugin)
resource "dokku_certs" "example_2" {
  global     = true
  cert       = "/etc/nginx/ssl/global.crt"
  key        = "/etc/nginx/ssl/global.key"
  depends_on = [dokku_plugin.global_cert]
}
