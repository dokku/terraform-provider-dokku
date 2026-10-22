data "dokku_service_property" "example" {
  service  = "postgres"
  name     = "my-db"
  property = "restart-policy"
}
