data "dokku_app_json_property" "example" {
  app      = "node-js-app"
  property = "appjson-path"
}
