package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/invers-gmbh/terraform-provider-pvescheduler/internal/provider"
)

func main() {
	err := providerserver.Serve(context.Background(), provider.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/invers-gmbh/pvescheduler",
	})
	if err != nil {
		log.Fatal(err)
	}
}
