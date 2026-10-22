resource "dokku_app" "hello_world" {
  app = "hello-world"
}

# Scale web and worker processes
resource "dokku_ps_scale" "example" {
  app = dokku_app.hello_world.app
  scale = {
    web    = 2
    worker = 1
  }
}
