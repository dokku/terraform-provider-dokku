data "dokku_builder_lambda_property" "example" {
  app      = "node-js-app"
  property = "lambdayml-path"
}
