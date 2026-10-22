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

# Add HTTP auth users to an app
resource "dokku_http_auth_user" "example" {
  app = dokku_app.hello_world.app
  users = [{
    password = "secret"
    username = "admin"
    }, {
    password = "hunter2"
    username = "ops"
  }]
  depends_on = [dokku_plugin.http_auth, dokku_http_auth.hello_world_http_auth]
}
