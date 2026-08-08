package node

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSecureSecretFileModeAllowsOnlyOwnerReadableFiles(t *testing.T) {
	require.True(t, secureSecretFileMode(0o400))
	require.True(t, secureSecretFileMode(0o600))
	for _, mode := range []os.FileMode{0o000, 0o200, 0o440, 0o640, 0o644, 0o700} {
		require.False(t, secureSecretFileMode(mode), "mode %04o", mode)
	}
}

func TestReadSecretFileFailsClosed(t *testing.T) {
	missing := "/run/secrets/stuhelper-test-secret-file-missing"
	_, err := readSecretFile(missing)
	require.Error(t, err)

	_, err = readSecretFile("/tmp/connector-password")
	require.Error(t, err)

	// Content validation and mode checks are exercised against a temporary
	// directory by the implementation-level helper below; the public reader
	// separately enforces the immutable /run/secrets reference root.
	directory := t.TempDir()
	valid := filepath.Join(directory, "valid")
	require.NoError(t, os.WriteFile(valid, []byte("secret-value\n"), 0o600))
	secret, err := readSecretFileFrom(valid)
	require.NoError(t, err)
	require.Equal(t, []byte("secret-value"), secret)
	wipe(secret)

	require.NoError(t, os.Chmod(valid, 0o644))
	_, err = readSecretFileFrom(valid)
	require.Error(t, err)

	symlink := filepath.Join(directory, "symlink")
	require.NoError(t, os.Symlink(valid, symlink))
	_, err = readSecretFileFrom(symlink)
	require.Error(t, err)

	multiline := filepath.Join(directory, "multiline")
	require.NoError(t, os.WriteFile(multiline, []byte("first\nsecond\n"), 0o600))
	_, err = readSecretFileFrom(multiline)
	require.Error(t, err)

	repeatedTrailingNewline := filepath.Join(directory, "repeated-trailing-newline")
	require.NoError(t, os.WriteFile(repeatedTrailingNewline, []byte("secret-value\n\n"), 0o600))
	_, err = readSecretFileFrom(repeatedTrailingNewline)
	require.Error(t, err)

	crlf := filepath.Join(directory, "crlf")
	require.NoError(t, os.WriteFile(crlf, []byte("secret-value\r\n"), 0o400))
	secret, err = readSecretFileFrom(crlf)
	require.NoError(t, err)
	require.Equal(t, []byte("secret-value"), secret)
	wipe(secret)
}

func TestValidateSecretParentDirectoriesRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	realDirectory := filepath.Join(root, "real")
	require.NoError(t, os.Mkdir(realDirectory, 0o700))
	require.NoError(t, validateSecretParentDirectories(root, filepath.Join(realDirectory, "secret")))

	symlinkDirectory := filepath.Join(root, "link")
	require.NoError(t, os.Symlink(realDirectory, symlinkDirectory))
	require.Error(t, validateSecretParentDirectories(root, filepath.Join(symlinkDirectory, "secret")))
}
