# Create a docker-local storage entry owned by the herokuish user
resource "dokku_storage_entry" "example" {
  name  = "node-js-app-data"
  chown = "herokuish"
}
