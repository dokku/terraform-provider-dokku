# Set chart values via a flat map of dotted paths
resource "dokku_scheduler_k3s_chart" "example" {
  chart = "ingress-nginx"
  values = {
    "controller.replicaCount"         = "3"
    "controller.resources.limits.cpu" = "200m"
  }
}

# Mix nested and flat; nested dotted leaves are escaped for Helm
resource "dokku_scheduler_k3s_chart" "example_2" {
  chart = "traefik"
  values = {
    "ports.web.redirectTo.port" = "websecure"
    service = {
      annotations = {
        "service.beta.kubernetes.io/aws-load-balancer-type" = "nlb"
      }
    }
  }
}
