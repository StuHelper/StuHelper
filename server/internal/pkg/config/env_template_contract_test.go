package config

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// These runtime aliases intentionally stay out of the templates. Operators
// should configure the canonical keys instead, so copying a template cannot
// accidentally suppress the fallback implemented by getEnv.
var acceptedUntemplatedRuntimeKeys = map[string]string{
	"AWS_CA_BUNDLE":    "standard AWS compatibility alias; use OBJECT_STORAGE_TLS_CA",
	"LOG_ENVIRONMENT":  "optional logging override; logs normally inherit APP_ENV",
	"LOG_SERVICE_NAME": "optional logging override; logs normally inherit OTEL_SERVICE_NAME",
}

func TestRuntimeEnvironmentKeysHaveTemplateCoverage(t *testing.T) {
	repoRoot := repositoryRoot(t)
	runtimeKeys := collectRuntimeEnvironmentKeys(
		t,
		filepath.Join(repoRoot, "server", "internal", "pkg", "config"),
	)
	templateKeys := collectEnvironmentTemplateKeys(
		t,
		filepath.Join(repoRoot, ".env.example"),
		filepath.Join(repoRoot, ".env.prod.example"),
	)

	var missing []string
	for key := range runtimeKeys {
		if _, documented := templateKeys[key]; documented {
			continue
		}
		if _, accepted := acceptedUntemplatedRuntimeKeys[key]; accepted {
			continue
		}
		missing = append(missing, key)
	}
	slices.Sort(missing)
	require.Empty(
		t,
		missing,
		"runtime config keys must appear in at least one operator template or in the narrow, explained alias allowlist",
	)

	for key, rationale := range acceptedUntemplatedRuntimeKeys {
		_, readAtRuntime := runtimeKeys[key]
		require.True(t, readAtRuntime, "remove stale runtime-key allowlist entry %s (%s)", key, rationale)
		_, templated := templateKeys[key]
		require.False(
			t,
			templated,
			"remove %s from the alias allowlist after adding it to an operator template (%s)",
			key,
			rationale,
		)
	}

	t.Logf(
		"covered %d runtime config keys with %d template keys and %d explained aliases",
		len(runtimeKeys),
		len(templateKeys),
		len(acceptedUntemplatedRuntimeKeys),
	)
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "resolve current test file")
	root := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "..", ".."))
	_, err := os.Stat(filepath.Join(root, "server", "go.mod"))
	require.NoError(t, err, "resolve repository root from %s", currentFile)
	return root
}

func collectRuntimeEnvironmentKeys(t *testing.T, configDir string) map[string]struct{} {
	t.Helper()
	entries, err := os.ReadDir(configDir)
	require.NoError(t, err)

	keys := make(map[string]struct{})
	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == "env.go" {
			continue
		}
		path := filepath.Join(configDir, name)
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		require.NoError(t, parseErr, "parse %s", path)

		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			getter, relevant := runtimeEnvironmentGetter(call.Fun)
			if !relevant {
				return true
			}
			require.NotEmpty(t, call.Args, "%s call in %s has no key argument", getter, path)
			literal, ok := call.Args[0].(*ast.BasicLit)
			require.True(
				t,
				ok && literal.Kind == token.STRING,
				"%s call in %s must use a string-literal key so template coverage remains auditable",
				getter,
				path,
			)
			key, unquoteErr := strconv.Unquote(literal.Value)
			require.NoError(t, unquoteErr, "decode environment key in %s", path)
			require.Regexp(t, `^[A-Z][A-Z0-9_]*$`, key, "invalid runtime environment key in %s", path)
			keys[key] = struct{}{}
			return true
		})
	}
	return keys
}

func runtimeEnvironmentGetter(expr ast.Expr) (string, bool) {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name, strings.HasPrefix(typed.Name, "getEnv")
	case *ast.SelectorExpr:
		pkg, ok := typed.X.(*ast.Ident)
		if !ok || pkg.Name != "os" || (typed.Sel.Name != "Getenv" && typed.Sel.Name != "LookupEnv") {
			return "", false
		}
		return fmt.Sprintf("%s.%s", pkg.Name, typed.Sel.Name), true
	default:
		return "", false
	}
}

func collectEnvironmentTemplateKeys(t *testing.T, paths ...string) map[string]struct{} {
	t.Helper()
	keys := make(map[string]struct{})
	for _, path := range paths {
		file, err := os.Open(path)
		require.NoError(t, err)

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, _, found := strings.Cut(line, "=")
			if found && isEnvironmentKey(key) {
				keys[key] = struct{}{}
			}
		}
		require.NoError(t, scanner.Err(), "read %s", path)
		require.NoError(t, file.Close(), "close %s", path)
	}
	return keys
}

func isEnvironmentKey(key string) bool {
	if key == "" || key[0] < 'A' || key[0] > 'Z' {
		return false
	}
	for _, character := range key[1:] {
		if (character < 'A' || character > 'Z') &&
			(character < '0' || character > '9') &&
			character != '_' {
			return false
		}
	}
	return true
}
