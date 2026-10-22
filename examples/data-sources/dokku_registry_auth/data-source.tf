data "dokku_registry_auth" "example" {
  app    = "node-js-app"
  server = "ghcr.io"
}
