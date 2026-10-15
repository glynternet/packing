package hosted

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func fixture(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(dir)
	require.NoError(t, err)
	require.NoError(t, s.Seed(map[string]string{"basics": "Tent\nWater\n"}, "ref: basics\n"))
	t.Cleanup(func() { s.Close() })
	return s, dir
}
func mutation(t Trip, kind string) Mutation {
	m := Mutation{ID: randomID(), Kind: kind, Epoch: t.Epoch, Generation: t.Generation, DraftRevision: t.DraftRevision, Text: t.Draft}
	if kind == "draft" {
		m.ETag = ETag(t, "draft", t.DraftRevision)
	} else {
		m.ETag = ETag(t, "snapshot", t.Generation)
	}
	return m
}
func pack(t Trip, name string, value bool) Mutation {
	m := mutation(t, "item")
	m.Item = hash([]byte(name))
	m.Packed = value
	m.ETag = ETag(t, "item-"+m.Item, t.Items[m.Item].Revision)
	return m
}
func code(t *testing.T, err error, status int) {
	t.Helper()
	var f *Fault
	require.ErrorAs(t, err, &f)
	require.Equal(t, status, f.Status)
}
func TestConcurrentItemsDraftAndApply(t *testing.T) {
	s, _ := fixture(t)
	_, err := s.Mutate(mutation(s.Read(), "apply"))
	require.NoError(t, err)
	base := s.Read()
	a, b := pack(base, "Tent", true), pack(base, "Water", true)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, m := range []Mutation{a, b} {
		wg.Add(1)
		go func(m Mutation) { defer wg.Done(); _, err := s.Mutate(m); errs <- err }(m)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.True(t, s.Read().Items[a.Item].Packed)
	require.True(t, s.Read().Items[b.Item].Packed)
	_, err = s.Mutate(pack(base, "Tent", false))
	code(t, err, 412)
	draft := mutation(base, "draft")
	draft.Text = "ref: basics\nCoat\n"
	_, err = s.Mutate(draft)
	require.NoError(t, err, "item edits must not invalidate drafts")
	before := s.Read()
	_, err = s.Mutate(mutation(before, "apply"))
	require.NoError(t, err)
	require.True(t, s.Read().Items[a.Item].Packed)
	require.False(t, s.Read().Items[hash([]byte("Coat"))].Packed)
	_, err = s.Mutate(pack(before, "Water", false))
	code(t, err, 409)
	// Removing/reintroducing an item must not resurrect old completion.
	draft = mutation(s.Read(), "draft")
	draft.Text = "Coat"
	_, err = s.Mutate(draft)
	require.NoError(t, err)
	_, err = s.Mutate(mutation(s.Read(), "apply"))
	require.NoError(t, err)
	draft = mutation(s.Read(), "draft")
	draft.Text = "Tent"
	_, err = s.Mutate(draft)
	require.NoError(t, err)
	_, err = s.Mutate(mutation(s.Read(), "apply"))
	require.NoError(t, err)
	require.False(t, s.Read().Items[a.Item].Packed)
}
func TestDraftInvalidTextAndPreconditions(t *testing.T) {
	s, _ := fixture(t)
	base := s.Read()
	m := mutation(base, "draft")
	m.Text = "bad: unfinished"
	_, err := s.Mutate(m)
	require.NoError(t, err)
	require.Equal(t, m.Text, s.Read().Draft)
	_, err = s.Mutate(mutation(s.Read(), "apply"))
	code(t, err, 422)
	_, err = s.Mutate(mutation(base, "draft"))
	code(t, err, 412)
	before := s.Read()
	m = mutation(before, "draft")
	m.ETag = ""
	_, err = s.Mutate(m)
	code(t, err, 428)
	m.ETag = "*"
	_, err = s.Mutate(m)
	code(t, err, 412)
	m = mutation(before, "draft")
	m.Epoch = "old"
	_, err = s.Mutate(m)
	code(t, err, 409)
	require.Equal(t, before, s.Read())
}
func TestReceiptRestartBackupRestore(t *testing.T) {
	s, dir := fixture(t)
	m := mutation(s.Read(), "apply")
	out, err := s.Mutate(m)
	require.NoError(t, err)
	_, err = s.Mutate(pack(s.Read(), "Tent", true))
	require.NoError(t, err)
	replay, err := s.Mutate(m)
	require.NoError(t, err)
	require.Equal(t, out, replay)
	require.Greater(t, s.Read().Revision, replay.Revision, "old acknowledgement must not replace aggregate")
	changed := m
	changed.Text = "another body"
	_, err = s.Mutate(changed)
	code(t, err, 409)
	_, err = Open(dir)
	require.Error(t, err, "a second writer must be refused")
	before := s.Read()
	require.NoError(t, s.Close())
	s, err = Open(dir)
	require.NoError(t, err)
	defer s.Close()
	require.Equal(t, before, s.Read())
	replay, err = s.Mutate(m)
	require.NoError(t, err)
	require.Equal(t, out, replay)
	backup := filepath.Join(t.TempDir(), "backup.json")
	require.NoError(t, s.Backup(backup))
	restored, err := Open(t.TempDir())
	require.NoError(t, err)
	defer restored.Close()
	require.NoError(t, restored.Restore(backup))
	after := restored.Read()
	require.NotEqual(t, before.Epoch, after.Epoch)
	after.Epoch = before.Epoch
	require.Equal(t, before, after)
	groups, err := restored.Preview("ref: basics")
	require.NoError(t, err)
	require.Len(t, groups, 1)
	_, err = restored.Mutate(m)
	code(t, err, 409)
	require.Error(t, restored.Restore(backup), "never overwrite an initialized store")
}
func TestFailedCommitAndInvalidBundles(t *testing.T) {
	s, dir := fixture(t)
	before := s.Read()
	m := mutation(before, "draft")
	m.Text = "replacement"
	s.replace = func(string, []byte) error { return errors.New("injected disk failure") }
	_, err := s.Mutate(m)
	code(t, err, 503)
	require.Equal(t, before, s.Read())
	require.False(t, s.Ready())
	s.Close()
	s, err = Open(dir)
	require.NoError(t, err)
	defer s.Close()
	require.Equal(t, before, s.Read())
	_, err = s.Mutate(m)
	require.NoError(t, err, "uncommitted mutation has no success receipt")
	for _, defs := range []map[string]string{{"../secret": "X"}, {"safe": "ref: ../secret"}, {"safe": "ref: safe"}} {
		require.Error(t, validateDefinitions(defs))
	}
	root := t.TempDir()
	require.NoError(t, os.Symlink("/etc/passwd", filepath.Join(root, "secret")))
	_, err = ReadDefinitions(root)
	require.Error(t, err)
}

func TestCommitBeforeLostAcknowledgement(t *testing.T) {
	s, dir := fixture(t)
	m := mutation(s.Read(), "apply")
	s.replace = func(path string, b []byte) error {
		if err := atomicWrite(path, b); err != nil {
			return err
		}
		return errors.New("injected failure after durable commit")
	}
	_, err := s.Mutate(m)
	code(t, err, 503)
	require.False(t, s.Ready())
	s.Close()
	s, err = Open(dir)
	require.NoError(t, err)
	defer s.Close()
	committed := s.Read()
	out, err := s.Mutate(m)
	require.NoError(t, err)
	require.Equal(t, committed.Revision, out.Revision)
	require.Equal(t, committed, s.Read())
}
func TestExpiredAccessAndRestoreRevocation(t *testing.T) {
	s, _ := fixture(t)
	secret, err := s.RotateAccess()
	require.NoError(t, err)
	token, err := s.Exchange(secret)
	require.NoError(t, err)
	require.True(t, s.Authorized(token))
	s.mu.Lock()
	s.data.Access.Sessions[hash([]byte(token))] = time.Now().Add(-time.Second)
	s.mu.Unlock()
	require.False(t, s.Authorized(token))
	backup := filepath.Join(t.TempDir(), "backup")
	require.NoError(t, s.Backup(backup))
	restored, err := Open(t.TempDir())
	require.NoError(t, err)
	defer restored.Close()
	require.NoError(t, restored.Restore(backup))
	_, err = restored.Exchange(secret)
	code(t, err, 401)
	require.False(t, restored.Authorized(token))
}
func TestReopenRejectsCorruptPinnedState(t *testing.T) {
	s, dir := fixture(t)
	s.Close()
	path := filepath.Join(dir, "state.json")
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	var d diskState
	require.NoError(t, json.Unmarshal(b, &d))
	d.Definitions["basics"] = "Changed"
	b, err = json.Marshal(d)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, b, 0600))
	_, err = Open(dir)
	require.ErrorContains(t, err, "checksum")
}
