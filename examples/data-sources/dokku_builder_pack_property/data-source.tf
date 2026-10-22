data "dokku_builder_pack_property" "example" {
  app      = "node-js-app"
  property = "projecttoml-path"
}
