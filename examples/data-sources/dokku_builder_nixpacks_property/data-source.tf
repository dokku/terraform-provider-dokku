data "dokku_builder_nixpacks_property" "example" {
  app      = "node-js-app"
  property = "nixpackstoml-path"
}
