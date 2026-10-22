data "dokku_network_property" "example" {
  app      = "hello-world"
  property = "attach-post-create"
}
