# Install a plugin from a git URL
resource "dokku_plugin" "example" {
  name = "redis"
  url  = "https://github.com/dokku/dokku-redis.git"
}

# Install a plugin pinned to a committish
resource "dokku_plugin" "example_2" {
  name       = "letsencrypt"
  url        = "https://github.com/dokku/dokku-letsencrypt.git"
  committish = "0.25.0"
}
