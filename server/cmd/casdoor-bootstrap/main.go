// Command casdoor-bootstrap ensures StuHelper-owned Casdoor bootstrap objects.
package main

import (
	"context"
	"log"
	"os"
	"time"

	"git.stuhelper.com/StuHelper/StuHelper/internal/platform/casdoor"
)

const bootstrapTimeout = 2 * time.Minute

func main() {
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
