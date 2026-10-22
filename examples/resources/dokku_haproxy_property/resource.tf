# Setting the letsencrypt email globally
resource "dokku_haproxy_property" "example" {
  global   = true
  property = "letsencrypt-email"
  value    = "admin@example.com"
}

# Setting the log level globally
resource "dokku_haproxy_property" "example_2" {
  global   = true
  property = "log-level"
  value    = "INFO"
}
