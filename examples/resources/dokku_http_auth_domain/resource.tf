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

resource "dokku_domains" "hello_world_domains" {
  app     = dokku_app.hello_world.app
  domains = ["app.example.com", "www.example.com"]
}

# Restrict HTTP auth to specific domains for an app
resource "dokku_http_auth_domain" "example" {
  app        = dokku_app.hello_world.app
  domains    = ["app.example.com", "www.example.com"]
  depends_on = [dokku_plugin.http_auth, dokku_http_auth.hello_world_http_auth, dokku_domains.hello_world_domains]
}
