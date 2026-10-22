resource "dokku_app" "hello_world" {
  app = "hello-world"
}

# Re-enable the zero downtime deployment (enabled by default)
resource "dokku_checks_toggle" "example" {
  app = dokku_app.hello_world.app
}
