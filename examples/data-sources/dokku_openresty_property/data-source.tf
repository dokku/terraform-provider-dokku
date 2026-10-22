data "dokku_openresty_property" "example" {
  app      = "node-js-app"
  property = "proxy-read-timeout"
}
