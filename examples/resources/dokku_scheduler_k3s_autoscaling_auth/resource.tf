resource "dokku_app" "node_js_app" {
  app = "node-js-app"
}

# Set AWS secret manager trigger metadata on an app
resource "dokku_scheduler_k3s_autoscaling_auth" "example" {
  app     = dokku_app.node_js_app.app
  trigger = "aws-secret-manager"
  metadata = {
    awsRegion  = "us-east-1"
    secretName = "my-secret"
  }
}

# Set a global trigger authentication shared across apps
resource "dokku_scheduler_k3s_autoscaling_auth" "example_2" {
  global  = true
  trigger = "aws-secret-manager"
  metadata = {
    awsRegion = "us-east-1"
    roleArn   = "arn:aws:iam::123456789012:role/keda"
  }
}
