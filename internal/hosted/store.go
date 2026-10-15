// Package hosted implements the single-trip HTTP MVP. SSE is deliberately absent.
package hosted

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/glynternet/packing/internal/load"
	"github.com/glynternet/packing/pkg/api"
	"github.com/glynternet/packing/pkg/list"
	"github.com/glynternet/pkg/log"
)

const MaxText = 64 << 10
const maxStore = 32 << 20
const maxReceipts = 10000
const ReceiptLifetime = 7 * 24 * time.Hour

var keyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)

type Fault struct {
	Status int
	Code   string
}

func (f *Fault) Error() string            { return f.Code }
func fault(status int, code string) error { return &Fault{status, code} }

type Snapshot struct {
	Selection string      `json:"selection"`
	Groups    []api.Group `json:"groups"`
}
type Item struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Packed   bool   `json:"packed"`
	Revision int    `json:"revision"`
}
type Trip struct {
	ID            string          `json:"id"`
	Epoch         string          `json:"epoch"`
	Revision      int             `json:"revision"`
	Library       string          `json:"library"`
	Draft         string          `json:"draft"`
	DraftRevision int             `json:"draftRevision"`
	Generation    int             `json:"generation"`
	Snapshot      Snapshot        `json:"snapshot"`
	Items         map[string]Item `json:"items"`
}
type Mutation struct {
	ID            string `json:"mutationId"`
	Kind          string `json:"kind"`
	ETag          string `json:"etag"`
	Epoch         string `json:"epoch"`
	Generation    int    `json:"generation"`
	DraftRevision int    `json:"draftRevision"`
	Text          string `json:"text"`
	Item          string `json:"item"`
	Packed        bool   `json:"packed"`
}

// Receipts contain compact original outcomes, never copies of the whole trip.
// A client reconciles the current aggregate after acknowledgement.
type Outcome struct {
	Revision      int `json:"revision"`
	DraftRevision int `json:"draftRevision"`
	Generation    int `json:"generation"`
}
type receipt struct {
	Hash    string
	At      time.Time
	Outcome Outcome
}
type diskState struct {
	Access      Access
	Schema      int
	Trip        Trip
	Definitions map[string]string
	Receipts    map[string]receipt
}
type Store struct {
	mu     sync.Mutex
	dir    string
	lock   *os.File
	data   diskState
	failed bool
	// replace is injectable for deterministic commit-failure tests.
	replace func(string, []byte) error
}

func randomID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func hash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func ETag(t Trip, resource string, revision int) string {
	return fmt.Sprintf(`"%s:%s:%d"`, t.Epoch, resource, revision)
}

// Open takes an OS advisory lock for the lifetime of the process. All operators,
// including backup/restore, use this lock; external file editing is unsupported.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("store directory must be private (0700), regular and not a symlink")
	}
	lock, err := os.OpenFile(filepath.Join(dir, "lock"), os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lock.Close()
		return nil, errors.New("store is already in use; stop its server first")
	}
	s := &Store{dir: dir, lock: lock, replace: atomicWrite}
	path := filepath.Join(dir, "state.json")
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		s.Close()
		return nil, err
	}
	defer file.Close()
	b, err := io.ReadAll(io.LimitReader(file, maxStore+1))
	if err == nil && len(b) > maxStore {
		err = errors.New("store too large")
	}
	if err == nil {
		err = json.Unmarshal(b, &s.data)
	}
	if err == nil {
		err = validateState(s.data)
	}
	if err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}
func (s *Store) Close() error { return s.lock.Close() }
func (s *Store) Ready() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.Schema == 1 && !s.failed
}
func (s *Store) Read() Trip {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, _ := json.Marshal(s.data.Trip)
	var t Trip
	_ = json.Unmarshal(b, &t)
	return t
}

// Seed only initializes an empty store. Definitions are copied once, never read
// from the source checkout while serving. An invalid draft is retained unapplied.
func (s *Store) Seed(definitions map[string]string, draft string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Schema != 0 {
		return errors.New("store already initialized")
	}
	if !validText(draft) {
		return errors.New("invalid or oversized UTF-8 draft")
	}
	if err := validateDefinitions(definitions); err != nil {
		return err
	}
	b, _ := json.Marshal(definitions)
	d := diskState{Schema: 1, Definitions: definitions, Receipts: map[string]receipt{}, Trip: Trip{
		ID: randomID()[:24], Epoch: randomID(), Revision: 1, Library: hash(b), Draft: draft, DraftRevision: 1, Generation: 1,
		Snapshot: Snapshot{Groups: []api.Group{}}, Items: map[string]Item{},
	}}
	return s.commit(d)
}
func validText(text string) bool { return len(text) <= MaxText && utf8.ValidString(text) }
func validKey(key string) bool   { return keyPattern.MatchString(key) && key != "." && key != ".." }
func validateDefinitions(defs map[string]string) error {
	if len(defs) > 512 {
		return errors.New("too many definitions (maximum 512)")
	}
	total := 0
	for k, v := range defs {
		total += len(v)
		if !validKey(k) || !validText(v) || total > 2<<20 {
			return errors.New("invalid definition key, text or bundle size")
		}
		if _, err := list.ParseContentsDefinition(strings.NewReader(v)); err != nil {
			return errors.New("invalid definition syntax")
		}
	}
	// Exercise every definition with the existing resolver, including cycles and
	// req semantics. A bounded bundle limits recursion and graph expansion.
	for key := range defs {
		if _, err := resolve(defs, "ref: "+key); err != nil {
			return errors.New("unresolvable definition bundle")
		}
	}
	return nil
}
func ReadDefinitions(dir string) (map[string]string, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	defs := map[string]string{}
	for _, entry := range entries {
		if !validKey(entry.Name()) || !entry.Type().IsRegular() {
			return nil, errors.New("definitions must be flat regular files with safe reference keys")
		}
		f, err := root.Open(entry.Name())
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(io.LimitReader(f, MaxText+1))
		f.Close()
		if err != nil {
			return nil, err
		}
		defs[entry.Name()] = string(b)
	}
	return defs, validateDefinitions(defs)
}

type bundle map[string]string

func (b bundle) GetContentsDefinition(key string) (api.Contents, error) {
	value, ok := b[key]
	if !validKey(key) || !ok {
		return api.Contents{}, errors.New("unknown reference")
	}
	return list.ParseContentsDefinition(strings.NewReader(value))
}
func resolve(defs map[string]string, text string) ([]api.Group, error) {
	if !validText(text) {
		return nil, errors.New("invalid text")
	}
	seed, err := list.ParseContentsDefinition(strings.NewReader(text))
	if err != nil {
		return nil, err
	}
	loader := load.Loader{ContentsDefinitionGetter: bundle(defs)}
	groups, err := loader.AllGroups(log.NewLogger(io.Discard), seed)
	if err != nil {
		return nil, err
	}
	if groups == nil {
		groups = []api.Group{}
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Name < groups[j].Name })
	count := 0
	for _, g := range groups {
		count += len(g.Contents.Items) + len(g.Contents.Refs)
	}
	if count > 10000 {
		return nil, errors.New("resolved list too large")
	}
	return groups, nil
}
func (s *Store) Preview(text string) ([]api.Group, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	groups, err := resolve(s.data.Definitions, text)
	if err != nil {
		return nil, fault(422, "cannot_resolve_selection")
	}
	return groups, nil
}
func (s *Store) Mutate(m Mutation) (Outcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failed || s.data.Schema != 1 {
		return Outcome{}, fault(503, "storage_unavailable")
	}
	if !keyPattern.MatchString(m.ID) || len(m.ID) < 16 {
		return Outcome{}, fault(400, "invalid_mutation_id")
	}
	payload, _ := json.Marshal(m)
	digest := hash(payload)
	if r, ok := s.data.Receipts[m.ID]; ok && time.Since(r.At) < ReceiptLifetime {
		if r.Hash != digest {
			return Outcome{}, fault(409, "mutation_id_reused")
		}
		return r.Outcome, nil
	}
	t := s.data.Trip
	if m.Epoch != t.Epoch {
		return Outcome{}, fault(409, "dataset_changed")
	}
	if m.ETag == "" {
		return Outcome{}, fault(428, "precondition_required")
	}
	var expected string
	switch m.Kind {
	case "draft":
		expected = ETag(t, "draft", t.DraftRevision)
	case "apply":
		expected = ETag(t, "snapshot", t.Generation)
	case "item":
		item, ok := t.Items[m.Item]
		if !ok {
			return Outcome{}, fault(409, "item_no_longer_present")
		}
		expected = ETag(t, "item-"+m.Item, item.Revision)
	default:
		return Outcome{}, fault(400, "unknown_mutation")
	}
	if m.Kind != "draft" && m.Generation != t.Generation {
		return Outcome{}, fault(409, "snapshot_changed")
	}
	if m.ETag != expected {
		return Outcome{}, fault(412, "changed_on_another_device")
	}
	// Work on a copy; no failed mutation can change the in-memory authority.
	b, _ := json.Marshal(s.data)
	var next diskState
	_ = json.Unmarshal(b, &next)
	n := &next.Trip
	switch m.Kind {
	case "draft":
		if !validText(m.Text) {
			return Outcome{}, fault(413, "invalid_or_oversized_text")
		}
		n.Draft = m.Text
		n.DraftRevision++
	case "apply":
		if m.DraftRevision != t.DraftRevision || m.Text != t.Draft {
			return Outcome{}, fault(412, "draft_changed")
		}
		groups, err := resolve(next.Definitions, t.Draft)
		if err != nil {
			return Outcome{}, fault(422, "cannot_apply_selection")
		}
		n.Snapshot = Snapshot{Selection: t.Draft, Groups: groups}
		n.Generation++
		n.Items = map[string]Item{}
		for _, g := range groups {
			for _, name := range g.Contents.Items {
				id := hash([]byte(name))
				old := t.Items[id]
				n.Items[id] = Item{ID: id, Name: name, Packed: old.Packed, Revision: old.Revision + 1}
			}
		}
	case "item":
		item := n.Items[m.Item]
		item.Packed = m.Packed
		item.Revision++
		n.Items[m.Item] = item
	}
	n.Revision++
	outcome := Outcome{n.Revision, n.DraftRevision, n.Generation}
	for key, r := range next.Receipts {
		if time.Since(r.At) >= ReceiptLifetime {
			delete(next.Receipts, key)
		}
	}
	if len(next.Receipts) >= maxReceipts {
		return Outcome{}, fault(503, "receipt_capacity_reached")
	}
	next.Receipts[m.ID] = receipt{digest, time.Now().UTC(), outcome}
	if err := s.commit(next); err != nil {
		return Outcome{}, fault(503, "storage_unavailable")
	}
	return outcome, nil
}
func (s *Store) commit(next diskState) error {
	b, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if len(b) > maxStore {
		return errors.New("store too large")
	}
	if err = s.replace(filepath.Join(s.dir, "state.json"), b); err != nil {
		// Rename may have succeeded before a directory-sync failure. Refuse further
		// reads/writes until restart reloads the durable authority and receipts.
		s.failed = true
		return err
	}
	s.data = next
	return nil
}
func atomicWrite(path string, b []byte) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".packing-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(f.Name(), path); err != nil {
		return err
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

// Backup must be run with the server stopped, taking the same lifetime lock.
// The single atomic image includes pinned definitions, trip and retry receipts.
func (s *Store) Backup(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failed || s.data.Schema != 1 {
		return errors.New("store unavailable")
	}
	b, err := json.Marshal(s.data)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err = f.Write(b); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	d, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

// Restore only accepts a new directory and rotates the epoch. Pending writes
// from the old dataset must be explicitly reviewed, never blindly replayed.
func (s *Store) Restore(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Schema != 0 {
		return errors.New("restore requires an empty store")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, maxStore+1))
	if err != nil {
		return err
	}
	if len(b) > maxStore {
		return errors.New("backup too large")
	}
	var d diskState
	if err = json.Unmarshal(b, &d); err != nil {
		return err
	}
	if err = validateState(d); err != nil {
		return err
	}
	d.Access = Access{}
	d.Trip.Epoch = randomID()
	d.Receipts = map[string]receipt{}
	return s.commit(d)
}

// Validate restored/reopened data before it can become serving authority.
func validateState(d diskState) error {
	if d.Schema != 1 || d.Trip.Epoch == "" || d.Trip.ID == "" || d.Receipts == nil || d.Definitions == nil || d.Trip.Items == nil || !validText(d.Trip.Draft) || d.Trip.Revision < 1 || d.Trip.DraftRevision < 1 || d.Trip.Generation < 1 {
		return errors.New("unsupported or invalid store")
	}
	if err := validateDefinitions(d.Definitions); err != nil {
		return err
	}
	definitions, _ := json.Marshal(d.Definitions)
	if hash(definitions) != d.Trip.Library {
		return errors.New("definition checksum mismatch")
	}
	groups, err := resolve(d.Definitions, d.Trip.Snapshot.Selection)
	if err != nil {
		return errors.New("invalid accepted snapshot")
	}
	expected, _ := json.Marshal(groups)
	actual, _ := json.Marshal(d.Trip.Snapshot.Groups)
	if string(expected) != string(actual) {
		return errors.New("accepted snapshot does not match pinned definitions")
	}
	items := map[string]string{}
	for _, g := range groups {
		for _, name := range g.Contents.Items {
			items[hash([]byte(name))] = name
		}
	}
	if len(items) != len(d.Trip.Items) {
		return errors.New("invalid packing progress")
	}
	for id, name := range items {
		item, ok := d.Trip.Items[id]
		if !ok || item.ID != id || item.Name != name || item.Revision < 1 {
			return errors.New("invalid packing item")
		}
	}
	return nil
}
