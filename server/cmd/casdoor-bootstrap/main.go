// Command casdoor-bootstrap ensures StuHelper-owned Casdoor bootstrap objects.
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/StuHelper/StuHelper/server/internal/platform/casdoor"
)

const bootstrapTimeout = 2 * time.Minute

func main() {
	if os.Getenv("CASDOOR_BOOTSTRAP_MODE") == "applications-only" {
		runApplicationBootstrap()
		return
	}

	settings, err := loadSettings(os.Getenv)
	if err != nil {
		log.Fatalf("invalid Casdoor bootstrap configuration: %v", err)
	}

	client, err := casdoor.NewBootstrapClient(settings.credential)
	if err != nil {
		log.Fatalf("create Casdoor bootstrap client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), bootstrapTimeout)
	defer cancel()
	if err := client.Bootstrap(ctx, settings.plan); err != nil {
		log.Fatalf("bootstrap Casdoor: %v", err)
	}

	log.Printf(
		"Casdoor bootstrap ensured organization=%s applications=%d roles=%d providers=%d",
		settings.plan.Organization.Name,
		len(settings.plan.Applications),
		len(settings.plan.Roles),
		len(settings.plan.Providers),
	)
}

func runApplicationBootstrap() {
	settings, err := loadApplicationBootstrapSettings(os.Getenv)
	if err != nil {
		log.Fatalf("invalid Casdoor application bootstrap configuration: %v", err)
	}

	client, err := casdoor.NewAppProvisioningClient(settings.credential)
	if err != nil {
		log.Fatalf("create Casdoor application provisioning client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), bootstrapTimeout)
	defer cancel()
	for _, app := range settings.applications {
		if err := client.EnsureApplication(ctx, app); err != nil {
			log.Fatalf("bootstrap Casdoor application %s: %v", app.Name, err)
		}
	}

	log.Printf(
		"Casdoor application bootstrap ensured applications=%d",
		len(settings.applications),
	)
}
