package file

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Selection stores raw selection text at a server-configured path. The parent
// directory must already exist. Concurrent saves use the last successful rename.
type Selection string

// Load creates an empty file if necessary, without truncating an existing file.
func (s Selection) Load() ([]byte, error) {
	path := string(s)
	data, err := os.ReadFile(path)
	if !errors.Is(err, os.ErrNotExist) {
		return data, err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if errors.Is(err, os.ErrExist) {
		// Another request created the file after our read.
		return os.ReadFile(path)
	}
	if err != nil {
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

// Save writes and syncs a temporary file in the destination directory, then
// renames it over the selection. Readers see either complete version. Existing
// permissions are preserved; new files are private to the server's user.
func (s Selection) Save(data []byte) error {
	path := string(s)
	mode := os.FileMode(0600)
	info, err := os.Lstat(path)
	if err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("selection path must be a regular file: %s", path)
		}
		mode = info.Mode().Perm()
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".packing-selection-*")
	if err != nil {
		return err
	}
	defer func() {
		// Close may already have succeeded; cleanup must not hide the save error.
		_ = f.Close()
		// After a successful rename the temporary name no longer exists.
		_ = os.Remove(f.Name())
	}()
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Chmod(mode); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
