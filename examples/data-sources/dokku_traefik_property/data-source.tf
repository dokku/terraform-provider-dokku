data "dokku_traefik_property" "example" {
  global   = true
  property = "letsencrypt-email"
}
