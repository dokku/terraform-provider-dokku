resource "dokku_plugin" "letsencrypt" {
  name = "letsencrypt"
  url  = "https://github.com/dokku/dokku-letsencrypt.git"
}

resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Setting the letsencrypt email for an app
resource "dokku_letsencrypt_property" "example" {
  app        = dokku_app.node_js_app.app
  property   = "email"
  value      = "admin@example.com"
  depends_on = [dokku_plugin.letsencrypt]
}

# Setting the dns provider for an app
resource "dokku_letsencrypt_property" "example_2" {
  app        = dokku_app.node_js_app.app
  property   = "dns-provider"
  value      = "namecheap"
  depends_on = [dokku_plugin.letsencrypt]
}

# Setting a dns-provider-* env var globally
resource "dokku_letsencrypt_property" "example_3" {
  global     = true
  property   = "dns-provider-NAMECHEAP_API_USER"
  value      = "deploy-bot"
  depends_on = [dokku_plugin.letsencrypt]
}
