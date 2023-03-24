package main

import (
	"context"
	"flag"
	"log"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// Provider documentation generation.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name hashicups

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/orcasecurity/orcasecurity",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), orcasecurity.New, opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
