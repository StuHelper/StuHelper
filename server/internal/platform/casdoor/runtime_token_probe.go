package casdoor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const defaultRuntimeTokenProbeTimeout = 30 * time.Second

// RuntimeTokenMinimizationProber runs a real OIDC token probe after a Casdoor
// third-party application has been created or updated.
type RuntimeTokenMinimizationProber interface {
	ProbeTokenMinimization(ctx context.Context, spec ApplicationSpec) (RuntimeTokenMinimizationProbeResult, error)
}

type RuntimeTokenProbeCommandConfig struct {
	Command string
	Issuer  string
	Timeout time.Duration
}

type CommandRuntimeTokenProber struct {
	command string
	issuer  string
	timeout time.Duration
}

type runtimeTokenProbeCommandInput struct {
	Issuer                 string   `json:"issuer"`
	CasdoorApplicationName string   `json:"casdoorApplicationName"`
	ClientID               string   `json:"clientID"`
	ClientSecret           string   `json:"clientSecret"`
	RedirectURIs           []string `json:"redirectURIs"`
	Scope                  string   `json:"scope"`
}

func NewCommandRuntimeTokenProber(cfg RuntimeTokenProbeCommandConfig) (*CommandRuntimeTokenProber, error) {
	command := strings.TrimSpace(cfg.Command)
	if command == "" {
		return nil, errorsNewTokenProbe("runtime token probe command is required")
	}
	issuer := strings.TrimSpace(cfg.Issuer)
	if issuer == "" {
		return nil, errorsNewTokenProbe("runtime token probe issuer is required")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultRuntimeTokenProbeTimeout
	}
	return &CommandRuntimeTokenProber{
		command: command,
		issuer:  issuer,
		timeout: timeout,
	}, nil
}

func (p *CommandRuntimeTokenProber) ProbeTokenMinimization(
	ctx context.Context,
	spec ApplicationSpec,
) (RuntimeTokenMinimizationProbeResult, error) {
	if p == nil {
		return RuntimeTokenMinimizationProbeResult{}, errorsNewTokenProbe("runtime token probe command is not configured")
	}
	redirectURI, err := runtimeTokenProbeRedirectURI(spec.RedirectURIs)
	if err != nil {
		return RuntimeTokenMinimizationProbeResult{}, err
	}
	clientID, err := runtimeTokenProbeClientID(spec.ClientID)
	if err != nil {
		return RuntimeTokenMinimizationProbeResult{}, err
	}
	clientSecret := strings.TrimSpace(spec.ClientSecret)
	payload, err := json.Marshal(runtimeTokenProbeCommandInput{
		Issuer:                 p.issuer,
		CasdoorApplicationName: strings.TrimSpace(spec.Name),
		ClientID:               clientID,
		ClientSecret:           clientSecret,
		RedirectURIs:           append([]string(nil), spec.RedirectURIs...),
		Scope:                  "openid",
	})
	if err != nil {
		return RuntimeTokenMinimizationProbeResult{}, fmt.Errorf("%w: build runtime probe input: %v",
			ErrTokenMinimizationProbeFailed, err)
	}

	runCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, p.command) // #nosec G204 -- p.command is an operator-configured, deployment-owned token probe runner; all untrusted data is passed via stdin/env.
	cmd.Stdin = bytes.NewReader(payload)
	cmd.Env = append(os.Environ(),
		"CASDOOR_ISSUER="+p.issuer,
		"CASDOOR_TOKEN_PROBE_CLIENT_ID="+clientID,
		"CASDOOR_TOKEN_PROBE_CLIENT_SECRET="+clientSecret,
		"CASDOOR_TOKEN_PROBE_REDIRECT_URI="+redirectURI,
		"CASDOOR_TOKEN_PROBE_SCOPE=openid",
		"CASDOOR_TOKEN_PROBE_OUTPUT=json",
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if runCtx.Err() != nil {
			return RuntimeTokenMinimizationProbeResult{}, fmt.Errorf("%w: runtime code-flow probe timed out after %s",
				ErrTokenMinimizationProbeFailed, p.timeout)
		}
		return RuntimeTokenMinimizationProbeResult{}, fmt.Errorf("%w: runtime code-flow probe command failed: %v: %s",
			ErrTokenMinimizationProbeFailed, err, truncateProbeOutput(stderr.String()+stdout.String()))
	}

	var result RuntimeTokenMinimizationProbeResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return RuntimeTokenMinimizationProbeResult{}, fmt.Errorf("%w: parse runtime code-flow probe evidence: %v: %s",
			ErrTokenMinimizationProbeFailed, err, truncateProbeOutput(stdout.String()))
	}
	return NormalizeRuntimeTokenMinimizationProbeResult(result)
}

func runtimeTokenProbeRedirectURI(redirectURIs []string) (string, error) {
	if len(redirectURIs) == 0 {
		return "", errorsNewTokenProbe("runtime code-flow probe redirect URI is required")
	}
	redirectURI := strings.TrimSpace(redirectURIs[0])
	if redirectURI == "" {
		return "", errorsNewTokenProbe("runtime code-flow probe redirect URI is required")
	}
	return redirectURI, nil
}

func runtimeTokenProbeClientID(clientID string) (string, error) {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return "", errorsNewTokenProbe("runtime code-flow probe client ID is required")
	}
	return clientID, nil
}

func truncateProbeOutput(value string) string {
	value = strings.TrimSpace(value)
	const maxProbeOutputBytes = 1024
	if len(value) <= maxProbeOutputBytes {
		return value
	}
	return value[:maxProbeOutputBytes] + "...[truncated]"
}

func errorsNewTokenProbe(message string) error {
	return fmt.Errorf("%w: %s", ErrTokenMinimizationProbeFailed, message)
}
