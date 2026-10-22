data "dokku_haproxy_property" "example" {
  global   = true
  property = "letsencrypt-email"
}
