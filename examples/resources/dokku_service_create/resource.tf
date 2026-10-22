resource "dokku_plugin" "redis" {
  name = "redis"
  url  = "https://github.com/dokku/dokku-redis.git"
}

# Create a redis service named my-redis
resource "dokku_service_create" "example" {
  service    = "redis"
  name       = "my-redis"
  depends_on = [dokku_plugin.redis]
}

resource "dokku_plugin" "postgres" {
  name = "postgres"
  url  = "https://github.com/dokku/dokku-postgres.git"
}

# Create a postgres service named my-db
resource "dokku_service_create" "example_2" {
  service    = "postgres"
  name       = "my-db"
  depends_on = [dokku_plugin.postgres]
}

# Create a redis service on a pinned image
resource "dokku_service_create" "example_3" {
  service       = "redis"
  name          = "my-pinned-redis"
  image         = "redis"
  image_version = "7.2.5"
  depends_on    = [dokku_plugin.redis]
}

# Reconcile a changed image pin by upgrading the service
resource "dokku_service_create" "example_4" {
  service       = "redis"
  name          = "my-upgraded-redis"
  image         = "redis"
  image_version = "7.2.5"
  image_drift   = "upgrade"
  depends_on    = [dokku_plugin.redis]
}

# Reconcile a changed shared memory size with set, stop and start
resource "dokku_service_create" "example_5" {
  service     = "redis"
  name        = "my-tuned-redis"
  shm_size    = "128m"
  image_drift = "upgrade"
  depends_on  = [dokku_plugin.redis]
}
