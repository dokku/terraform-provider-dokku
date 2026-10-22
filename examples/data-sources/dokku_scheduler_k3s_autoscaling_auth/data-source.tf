data "dokku_scheduler_k3s_autoscaling_auth" "example" {
  app     = "node-js-app"
  trigger = "aws-secret-manager"
}
