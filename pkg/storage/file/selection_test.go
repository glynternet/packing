package file_test

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/glynternet/packing/pkg/storage/file"
	"github.com/stretchr/testify/require"
)

func TestSelectionLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trip")
	selection := file.Selection(path)
	data, err := selection.Load()
	require.NoError(t, err)
	require.Empty(t, data)
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())

	text := "# trip\nref: sun\n\nPassport\n"
	require.NoError(t, os.WriteFile(path, []byte(text), 0600))
	data, err = selection.Load()
	require.NoError(t, err)
	require.Equal(t, text, string(data))

	_, err = file.Selection(filepath.Join(path, "missing")).Load()
	require.Error(t, err)
}

func TestSelectionSaveReplacesFileAndPreservesPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trip")
	require.NoError(t, os.WriteFile(path, []byte("original"), 0600))
	require.NoError(t, os.Chmod(path, 0640))
	old, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, old.Close()) })

	selection := file.Selection(path)
	text := "# preserve spacing\n  ref: sun\nreq: warm\nPassport\n"
	require.NoError(t, selection.Save([]byte(text)))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, text, string(data))
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0640), info.Mode().Perm())
	// An existing reader still sees the old inode; saves never truncate it.
	buffer := make([]byte, 8)
	_, err = old.Read(buffer)
	require.NoError(t, err)
	require.Equal(t, "original", string(buffer))
	require.NoError(t, selection.Save(nil))
	data, err = os.ReadFile(path)
	require.NoError(t, err)
	require.Empty(t, data)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1, "temporary files must be cleaned up")
}

func TestSelectionSaveFailurePreservesDestination(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trip")
	require.NoError(t, os.Mkdir(path, 0700))
	sentinel := filepath.Join(path, "keep")
	require.NoError(t, os.WriteFile(sentinel, []byte("original"), 0600))
	require.Error(t, file.Selection(path).Save([]byte("replacement")))
	data, err := os.ReadFile(sentinel)
	require.NoError(t, err)
	require.Equal(t, "original", string(data))

	link := filepath.Join(dir, "link")
	require.NoError(t, os.Symlink(sentinel, link))
	require.Error(t, file.Selection(link).Save([]byte("replacement")))
	data, err = os.ReadFile(sentinel)
	require.NoError(t, err)
	require.Equal(t, "original", string(data))
	require.Error(t, file.Selection(filepath.Join(dir, "missing", "trip")).Save(nil))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 2)
}

func TestConcurrentSelectionSavesExposeCompleteFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trip")
	selection := file.Selection(path)
	first, second := strings.Repeat("A", 128*1024), strings.Repeat("B", 128*1024)
	require.NoError(t, selection.Save([]byte(first)))
	var workers sync.WaitGroup
	for _, text := range []string{first, second} {
		workers.Go(func() {
			for range 10 {
				if err := selection.Save([]byte(text)); err != nil {
					t.Error(err)
					return
				}
			}
		})
	}
	t.Cleanup(workers.Wait)
	for range 100 {
		data, err := selection.Load()
		require.NoError(t, err)
		require.True(t, string(data) == first || string(data) == second, "read a partial selection")
	}
	workers.Wait()
}
