resource "dokku_plugin" "http_auth" {
  name = "http-auth"
  url  = "https://github.com/dokku/dokku-http-auth.git"
}

resource "dokku_app" "hello_world" {
  app = "hello-world"
}

resource "dokku_http_auth" "hello_world_http_auth" {
  app        = dokku_app.hello_world.app
  depends_on = [dokku_plugin.http_auth]
}

# Allow IP addresses to bypass HTTP auth for an app
resource "dokku_http_auth_allowed_ip" "example" {
  app         = dokku_app.hello_world.app
  allowed_ips = ["192.0.2.1", "198.51.100.0/24"]
  depends_on  = [dokku_plugin.http_auth, dokku_http_auth.hello_world_http_auth]
}
