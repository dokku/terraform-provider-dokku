resource "dokku_plugin" "postgres" {
  name = "postgres"
  url  = "https://github.com/dokku/dokku-postgres.git"
}

resource "dokku_service_create" "postgres_my_db" {
  service    = "postgres"
  name       = "my-db"
  depends_on = [dokku_plugin.postgres]
}

# Set the restart-policy for a postgres service
resource "dokku_service_property" "example" {
  service    = dokku_service_create.postgres_my_db.service
  name       = dokku_service_create.postgres_my_db.name
  property   = "restart-policy"
  value      = "always"
  depends_on = [dokku_plugin.postgres]
}

# Publish an exposed postgres service directly rather than through an ambassador
resource "dokku_service_property" "example_2" {
  service    = dokku_service_create.postgres_my_db.service
  name       = dokku_service_create.postgres_my_db.name
  property   = "expose-mode"
  value      = "direct"
  depends_on = [dokku_plugin.postgres]
}
