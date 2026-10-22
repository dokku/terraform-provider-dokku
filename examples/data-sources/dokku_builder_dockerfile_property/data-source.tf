data "dokku_builder_dockerfile_property" "example" {
  app      = "node-js-app"
  property = "dockerfile-path"
}
