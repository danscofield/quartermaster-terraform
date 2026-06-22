package main

import (
	"context"
	"log"

	"github.com/dscof/terraform-provider-quartermaster/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	err := providerserver.Serve(context.Background(), provider.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/dscof/quartermaster",
	})
	if err != nil {
		log.Fatal(err)
	}
}
