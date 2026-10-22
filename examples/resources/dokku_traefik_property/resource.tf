# Setting the letsencrypt email globally
resource "dokku_traefik_property" "example" {
  global   = true
  property = "letsencrypt-email"
  value    = "admin@example.com"
}

# Setting the log level globally
resource "dokku_traefik_property" "example_2" {
  global   = true
  property = "log-level"
  value    = "INFO"
}

# Setting a dns-provider-* env var globally
resource "dokku_traefik_property" "example_3" {
  global   = true
  property = "dns-provider-CLOUDFLARE_DNS_API_TOKEN"
  value    = "cf-token"
}
