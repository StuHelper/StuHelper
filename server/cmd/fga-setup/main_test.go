package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAuthorizationModelJSONPath(t *testing.T) {
	dir := t.TempDir()
	fgaPath := filepath.Join(dir, "model.fga")
	jsonPath := filepath.Join(dir, "model.json")
	if err := os.WriteFile(fgaPath, []byte("model\n  schema 1.1\n"), 0o600); err != nil {
		t.Fatalf("write fga model: %v", err)
	}
	if err := os.WriteFile(jsonPath, []byte(`{"schema_version":"1.1","type_definitions":[{"type":"user"}]}`), 0o600); err != nil {
		t.Fatalf("write json model: %v", err)
	}

	got, err := authorizationModelJSONPath(fgaPath)
	if err != nil {
		t.Fatalf("authorizationModelJSONPath(.fga) returned error: %v", err)
	}
	if got != jsonPath {
		t.Fatalf("authorizationModelJSONPath(.fga) = %q, want %q", got, jsonPath)
	}

	got, err = authorizationModelJSONPath(jsonPath)
	if err != nil {
		t.Fatalf("authorizationModelJSONPath(.json) returned error: %v", err)
	}
	if got != jsonPath {
		t.Fatalf("authorizationModelJSONPath(.json) = %q, want %q", got, jsonPath)
	}
}

func TestAuthorizationModelJSONPathRejectsMissingCompanion(t *testing.T) {
	dir := t.TempDir()
	fgaPath := filepath.Join(dir, "model.fga")
	if err := os.WriteFile(fgaPath, []byte("model\n  schema 1.1\n"), 0o600); err != nil {
		t.Fatalf("write fga model: %v", err)
	}

	if _, err := authorizationModelJSONPath(fgaPath); err == nil {
		t.Fatal("expected missing companion error")
	}
}

func TestResolveModelPathAllowsRepoRootWhenRunFromServer(t *testing.T) {
	root := t.TempDir()
	serverDir := filepath.Join(root, "server")
	modelPath := filepath.Join(root, "infra", "openfga", "model.fga")
	if err := os.MkdirAll(filepath.Dir(modelPath), 0o700); err != nil {
		t.Fatalf("mkdir model dir: %v", err)
	}
	if err := os.MkdirAll(serverDir, 0o700); err != nil {
		t.Fatalf("mkdir server dir: %v", err)
	}
	if err := os.WriteFile(modelPath, []byte("model\n  schema 1.1\n"), 0o600); err != nil {
		t.Fatalf("write model: %v", err)
	}

	t.Chdir(serverDir)
	got, err := resolveModelPath(modelPath)
	if err != nil {
		t.Fatalf("resolveModelPath returned error: %v", err)
	}
	if got != modelPath {
		t.Fatalf("resolveModelPath = %q, want %q", got, modelPath)
	}
}

func TestResolveModelPathRejectsOutsideWorkspace(t *testing.T) {
	root := t.TempDir()
	serverDir := filepath.Join(root, "repo", "server")
	outsidePath := filepath.Join(root, "outside", "model.fga")
	if err := os.MkdirAll(filepath.Dir(outsidePath), 0o700); err != nil {
		t.Fatalf("mkdir outside dir: %v", err)
	}
	if err := os.MkdirAll(serverDir, 0o700); err != nil {
		t.Fatalf("mkdir server dir: %v", err)
	}
	if err := os.WriteFile(outsidePath, []byte("model\n  schema 1.1\n"), 0o600); err != nil {
		t.Fatalf("write outside model: %v", err)
	}

	t.Chdir(serverDir)
	if _, err := resolveModelPath(outsidePath); err == nil {
		t.Fatal("expected workspace escape error")
	}
}

func TestParseAuthorizationModelJSON(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "..", "infra", "openfga", "model.json"))
	if err != nil {
		t.Fatalf("read model json: %v", err)
	}

	model, err := parseAuthorizationModelJSON(content)
	if err != nil {
		t.Fatalf("parseAuthorizationModelJSON returned error: %v", err)
	}
	if model.SchemaVersion != "1.1" {
		t.Fatalf("schema version = %q, want 1.1", model.SchemaVersion)
	}
	if len(model.TypeDefinitions) == 0 {
		t.Fatal("expected type definitions")
	}
}

func TestParseAuthorizationModelJSONRejectsEmptyModel(t *testing.T) {
	_, err := parseAuthorizationModelJSON([]byte(`{"schema_version":"1.1","type_definitions":[]}`))
	if err == nil {
		t.Fatal("expected empty model error")
	}
}
