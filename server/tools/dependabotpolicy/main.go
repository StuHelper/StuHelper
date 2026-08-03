// Command dependabotpolicy enforces develop-first routing for every Dependabot
// version-update entry using a real YAML parser.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

type dependabotConfig struct {
	Version int                `yaml:"version"`
	Updates []dependabotUpdate `yaml:"updates"`
}

type dependabotUpdate struct {
	PackageEcosystem string `yaml:"package-ecosystem"`
	TargetBranch     string `yaml:"target-branch"`
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[dependabot-policy][error] "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	configPath := flag.String("config", "", "path to .github/dependabot.yml")
	flag.Parse()
	if *configPath == "" {
		fail("--config is required")
	}

	configFile, err := os.Open(*configPath)
	if err != nil {
		fail("open %s: %v", *configPath, err)
	}
	defer configFile.Close()

	decoder := yaml.NewDecoder(configFile)
	var config dependabotConfig
	if err := decoder.Decode(&config); err != nil {
		fail("parse %s: %v", *configPath, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err != nil {
			fail("parse trailing YAML document in %s: %v", *configPath, err)
		}
		fail("%s must contain exactly one YAML document", *configPath)
	}

	if config.Version != 2 {
		fail("Dependabot version must be 2")
	}
	if len(config.Updates) == 0 {
		fail("Dependabot updates must contain at least one version-update entry")
	}
	for index, update := range config.Updates {
		ecosystem := update.PackageEcosystem
		if ecosystem == "" {
			fail("updates[%d] must set package-ecosystem", index)
		}
		if update.TargetBranch != "develop" {
			fail("%s must set target-branch: develop", ecosystem)
		}
	}

	fmt.Println("[dependabot-policy] all version-update ecosystems target develop")
}
