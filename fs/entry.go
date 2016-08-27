package fs

import (
	"encoding/json"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Entry represents a filesystem entry, which can be Directory, File, or Symlink
type Entry interface {
	Parent() Directory
	Metadata() *EntryMetadata
}

// EntryType is a type of a filesystem entry.
type EntryType string

// Supported entry types.
const (
	EntryTypeUnknown   EntryType = ""  // unknown type
	EntryTypeFile      EntryType = "f" // file
	EntryTypeDirectory EntryType = "d" // directory
	EntryTypeSymlink   EntryType = "s" // symbolic link
)

// Permissions encapsulates UNIX permissions for a filesystem entry.
type Permissions int

// MarshalJSON emits permissions as octal string.
func (p Permissions) MarshalJSON() ([]byte, error) {
	if p == 0 {
		return nil, nil
	}

	s := "0" + strconv.FormatInt(int64(p), 8)
	return json.Marshal(&s)
}

// UnmarshalJSON parses octal permissions string from JSON.
func (p *Permissions) UnmarshalJSON(b []byte) error {
	var s string

	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	v, err := strconv.ParseInt(s, 0, 32)
	if err != nil {
		return err
	}

	*p = Permissions(v)
	return nil
}

// EntryMetadata stores attributes of a single entry in a directory.
type EntryMetadata struct {
	Name        string      `json:"name,omitempty"`
	Type        EntryType   `json:"type,omitempty"`
	Permissions Permissions `json:"mode,omitempty"`
	FileSize    int64       `json:"size,omitempty"`
	ModTime     time.Time   `json:"mtime,omitempty"`
	UserID      uint32      `json:"uid,omitempty"`
	GroupID     uint32      `json:"gid,omitempty"`
}

// FileMode returns os.FileMode corresponding to Type and Permissions of the entry metadata.
func (e *EntryMetadata) FileMode() os.FileMode {
	perm := os.FileMode(e.Permissions)

	switch e.Type {
	default:
		return perm

	case EntryTypeFile:
		return perm

	case EntryTypeDirectory:
		return perm | os.ModeDir

	case EntryTypeSymlink:
		return perm | os.ModeSymlink
	}
}

// Entries is a list of entries sorted by name.
type Entries []Entry

// Reader allows reading from a file and retrieving its metadata.
type Reader interface {
	io.ReadCloser
	EntryMetadata() (*EntryMetadata, error)
}

// File represents an entry that is a file.
type File interface {
	Entry
	Open() (Reader, error)
}

// Directory represents contents of a directory.
type Directory interface {
	Entry
	Readdir() (Entries, error)
}

// Symlink represents a symbolic link entry.
type Symlink interface {
	Entry
	Readlink() (string, error)
}

type entry struct {
	parent   Directory
	metadata *EntryMetadata
}

func (e *entry) Parent() Directory {
	return e.parent
}

func (e *entry) Metadata() *EntryMetadata {
	return e.metadata
}

func newEntry(md *EntryMetadata, parent Directory) entry {
	return entry{parent, md}
}

type sortedEntries Entries

func (e sortedEntries) Len() int      { return len(e) }
func (e sortedEntries) Swap(i, j int) { e[i], e[j] = e[j], e[i] }
func (e sortedEntries) Less(i, j int) bool {
	return e[i].Metadata().Name < e[j].Metadata().Name
}

// FindByName returns an entry with a given name, or nil if not found.
func (e Entries) FindByName(n string) Entry {
	i := sort.Search(
		len(e),
		func(i int) bool {
			return e[i].Metadata().Name >= n
		},
	)
	if i < len(e) && e[i].Metadata().Name == n {
		return e[i]
	}

	return nil
}

// EntryPath returns a path of a given entry from its root node.
func EntryPath(e Entry) string {
	var parts []string

	p := e
	for p != nil {
		parts = append(parts, p.Metadata().Name)
		p = p.Parent()
	}

	// Invert parts
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}

	return strings.Join(parts, "/")
}
