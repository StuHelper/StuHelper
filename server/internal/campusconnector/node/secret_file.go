package node

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	secretFileRoot         = "/run/secrets"
	maximumSecretFileBytes = 16 * 1024
)

// readSecretFile reads a single credential from the connector's read-only
// secret mount. The reference is validated before open, symlinks and
// non-regular files are rejected, and group/other permission bits fail closed.
func readSecretFile(path string) ([]byte, error) {
	if !validSecretFileReference(path) {
		return nil, errors.New("secret file reference is invalid")
	}
	if err := validateSecretParentDirectories(secretFileRoot, path); err != nil {
		return nil, err
	}
	return readSecretFileFrom(path)
}

// validateSecretParentDirectories rejects symlinks in every path component
// below the fixed mount. Lstat on the leaf alone is insufficient because a
// symlinked parent could otherwise turn a lexically safe reference into an
// arbitrary local-file read.
func validateSecretParentDirectories(root, path string) error {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == "." || relative == ".." ||
		filepath.IsAbs(relative) ||
		len(relative) > 2 && relative[:3] == ".."+string(os.PathSeparator) {
		return errors.New("secret file reference is invalid")
	}
	for current := filepath.Dir(path); ; current = filepath.Dir(current) {
		info, statErr := os.Lstat(current)
		if statErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("secret file parent directory is unavailable")
		}
		if current == root {
			return nil
		}
		if parent := filepath.Dir(current); parent == current {
			return errors.New("secret file reference is invalid")
		}
	}
}

func readSecretFileFrom(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, errors.New("secret file is unavailable")
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("secret file must be a regular file")
	}
	if !secureSecretFileMode(info.Mode()) {
		return nil, errors.New("secret file permissions must be 0400 or 0600")
	}
	// #nosec G304 -- path is constrained to /run/secrets by
	// validSecretFileReference and is never supplied by a remote request.
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.New("secret file is unavailable")
	}
	openedInfo, err := file.Stat()
	if err != nil || !openedInfo.Mode().IsRegular() || !secureSecretFileMode(openedInfo.Mode()) ||
		!os.SameFile(info, openedInfo) {
		if closeErr := file.Close(); closeErr != nil {
			return nil, errors.New("secret file could not be closed")
		}
		return nil, errors.New("secret file changed while opening")
	}
	raw, readErr := io.ReadAll(io.LimitReader(file, maximumSecretFileBytes+1))
	closeErr := file.Close()
	if readErr != nil {
		wipe(raw)
		return nil, errors.New("secret file could not be read")
	}
	if closeErr != nil {
		wipe(raw)
		return nil, errors.New("secret file could not be closed")
	}
	if len(raw) > maximumSecretFileBytes {
		wipe(raw)
		return nil, fmt.Errorf("secret file exceeds %d bytes", maximumSecretFileBytes)
	}
	if bytes.HasSuffix(raw, []byte("\r\n")) {
		raw = raw[:len(raw)-2]
	} else if bytes.HasSuffix(raw, []byte("\n")) {
		raw = raw[:len(raw)-1]
	}
	if len(raw) == 0 || bytes.IndexByte(raw, 0) >= 0 ||
		bytes.IndexByte(raw, '\n') >= 0 || bytes.IndexByte(raw, '\r') >= 0 {
		wipe(raw)
		return nil, errors.New("secret file must contain one non-empty line")
	}
	return raw, nil
}

func secureSecretFileMode(mode os.FileMode) bool {
	permissions := mode.Perm()
	return permissions&0o400 != 0 && permissions&0o177 == 0
}

func secretFileAvailable(path string) bool {
	secret, err := readSecretFile(path)
	if secret != nil {
		wipe(secret)
	}
	return err == nil
}
