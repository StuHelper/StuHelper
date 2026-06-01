package user

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestQQBindingNicknameArtifactsRemoved(t *testing.T) {
	t.Parallel()

	files := []string{
		serverFilePath("api", "components", "schemas", "user-system.yaml"),
		serverFilePath("api", "paths", "user-identity.yaml"),
		serverFilePath("api", "openapi.yaml"),
		serverFilePath("api", "openapi.bundled.yaml"),
		serverFilePath("internal", "api", "gen", "server.gen.go"),
		repoFilePath("clients", "shared", "src", "types", "api.gen.ts"),
		repoFilePath("clients", "shared", "src", "api", "identity.ts"),
		repoFilePath("clients", "web", "src", "modules", "user", "views", "QQBindingPage.vue"),
		repoFilePath("clients", "web", "src", "modules", "user", "views", "AccountProfilePage.vue"),
		repoFilePath("clients", "web", "src", "modules", "user", "views", "ProfileSection.vue"),
		serverFilePath("migrations", "000001_initial_schema.up.sql"),
	}
	for _, file := range files {
		source := readQQBindingContractFile(t, file)
		for _, forbidden := range []string{"qqNickname", "QQNickname", "qq_nickname"} {
			if strings.Contains(source, forbidden) {
				t.Fatalf("%s must not contain removed QQ nickname field %q", file, forbidden)
			}
		}
	}
}

func TestQQBindingNicknameDropMigrationExists(t *testing.T) {
	t.Parallel()

	source := readQQBindingContractFile(t, serverFilePath("migrations", "000002_drop_qq_nickname_columns.up.sql"))
	for _, expected := range []string{
		"ALTER TABLE IF EXISTS public.group_admission_sessions",
		"ALTER TABLE IF EXISTS public.user_qq_bindings",
		"DROP COLUMN IF EXISTS qq_nickname",
	} {
		if !strings.Contains(source, expected) {
			t.Fatalf("QQ nickname drop migration missing %q", expected)
		}
	}
}

func readQQBindingContractFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}

func serverFilePath(parts ...string) string {
	base := []string{"..", "..", ".."}
	return filepath.Join(append(base, parts...)...)
}

func repoFilePath(parts ...string) string {
	base := []string{"..", "..", "..", ".."}
	return filepath.Join(append(base, parts...)...)
}
