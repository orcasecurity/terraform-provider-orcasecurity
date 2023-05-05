package main

//go:generate terraform fmt -recursive ./examples/
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name orcasecurity

import (
	"context"
	"flag"
	"log"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/getsentry/sentry-go"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

const sentryDSN = "https://6a305d50bdb04e0aa709c8f2f24dec03@o482658.ingest.sentry.io/4505086124752896"

var (
	// these will be set by the goreleaser configuration
	// to appropriate values for the compiled binary.
	version string = "dev"

	// goreleaser can pass other information to the main package, such as the specific commit
	// https://goreleaser.com/cookbooks/using-main.version/
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	err := sentry.Init(sentry.ClientOptions{
		Dsn:   sentryDSN,
		Debug: debug,
	})
	if err != nil {
		log.Printf("Sentry.Init failure: %s", err)
	}

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/orcasecurity/orcasecurity",
		Debug:   debug,
	}

	err = providerserver.Serve(context.Background(), orcasecurity.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
