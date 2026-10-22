data "dokku_registry_property" "example" {
  app      = "node-js-app"
  property = "image-repo"
}
