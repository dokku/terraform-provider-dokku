data "dokku_nginx_property" "example" {
  app      = "node-js-app"
  property = "proxy-read-timeout"
}
