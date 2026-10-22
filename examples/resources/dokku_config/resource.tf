resource "dokku_app" "hello_world" {
  app = "hello-world"
}

# set KEY=VALUE
resource "dokku_config" "example" {
  app     = dokku_app.hello_world.app
  restart = true
  config = {
    KEY = "VALUE_1"
  }
}
