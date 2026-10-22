package main

import (
	"context"
	"flag"
	"log"

	"github.com/dokku/terraform-provider-dokku/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

//go:generate go run ./tools/examples
//go:generate go tool tfplugindocs generate --provider-name dokku

// version is set by goreleaser.
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run the provider with support for debuggers such as delve")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/dokku/dokku",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err.Error())
	}
}
