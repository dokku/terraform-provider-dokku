data "dokku_builder_railpack_property" "example" {
  app      = "node-js-app"
  property = "railpackjson-path"
}
