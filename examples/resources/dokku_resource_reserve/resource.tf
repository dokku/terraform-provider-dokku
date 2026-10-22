resource "dokku_app" "hello_world" {
  app = "hello-world"
}

# Set CPU and memory reservations
resource "dokku_resource_reserve" "example" {
  app = dokku_app.hello_world.app
  resources = {
    cpu    = "100"
    memory = "256"
  }
}

# Set memory reservation for web process type
resource "dokku_resource_reserve" "example_2" {
  app          = dokku_app.hello_world.app
  process_type = "web"
  resources = {
    memory = "512"
  }
}
