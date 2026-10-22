resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Setting the lambda.yml path for an app
resource "dokku_builder_lambda_property" "example" {
  app      = dokku_app.node_js_app.app
  property = "lambdayml-path"
  value    = "config/lambda.yml"
}

# Setting the lambda.yml path globally
resource "dokku_builder_lambda_property" "example_2" {
  global   = true
  property = "lambdayml-path"
  value    = "lambda.yml"
}
