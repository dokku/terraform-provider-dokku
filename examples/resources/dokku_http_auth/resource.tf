resource "dokku_plugin" "http_auth" {
  name = "http-auth"
  url  = "https://github.com/dokku/dokku-http-auth.git"
}

resource "dokku_app" "hello_world" {
  app = "hello-world"
}

# Enable HTTP authentication for an app
resource "dokku_http_auth" "example" {
  app        = dokku_app.hello_world.app
  username   = "admin"
  password   = "secret"
  depends_on = [dokku_plugin.http_auth]
}
