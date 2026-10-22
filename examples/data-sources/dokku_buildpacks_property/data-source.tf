data "dokku_buildpacks_property" "example" {
  app      = "node-js-app"
  property = "stack"
}
