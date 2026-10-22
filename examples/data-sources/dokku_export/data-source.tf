# Export one app as a docket recipe.
data "dokku_export" "api" {
  apps   = ["api"]
  format = "yaml"
}

output "recipe" {
  value = data.dokku_export.api.recipe
}
