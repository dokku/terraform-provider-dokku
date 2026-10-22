resource "dokku_app" "hello_world" {
  app = "hello-world"
}

resource "dokku_network" "example_network" {
  name = "example-network"
}

# Associates a network after a container is created but before it is started
resource "dokku_network_property" "example" {
  app        = dokku_app.hello_world.app
  property   = "attach-post-create"
  value      = "example-network"
  depends_on = [dokku_network.example_network]
}

# Associates the network at container creation
resource "dokku_network_property" "example_2" {
  app        = dokku_app.hello_world.app
  property   = "initial-network"
  value      = "example-network"
  depends_on = [dokku_network.example_network]
}

# Setting a global network property
resource "dokku_network_property" "example_3" {
  global     = true
  property   = "attach-post-create"
  value      = "example-network"
  depends_on = [dokku_network.example_network]
}
