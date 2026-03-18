package rootfs

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestRootFSOpenWithinRoot(t *testing.T) {
	fsys := fstest.MapFS{
		"cfg/mapper.xml": {Data: []byte("ok")},
	}

	file, err := New(fsys, "cfg").Open("mapper.xml")
	if err != nil {
		t.Fatalf("Open(mapper.xml) error = %v", err)
	}
	_ = file.Close()
}

func TestRootFSRejectsEscapingPath(t *testing.T) {
	fsys := fstest.MapFS{
		"cfg/mapper.xml":  {Data: []byte("ok")},
		"secret/evil.xml": {Data: []byte("nope")},
	}

	_, err := New(fsys, "cfg").Open("../secret/evil.xml")
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("Open(../secret/evil.xml) error = %v, want permission error", err)
	}
}

func TestRootFSRejectsAbsolutePath(t *testing.T) {
	fsys := fstest.MapFS{
		"cfg/mapper.xml": {Data: []byte("ok")},
	}

	_, err := New(fsys, "cfg").Open("/secret/evil.xml")
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("Open(/secret/evil.xml) error = %v, want permission error", err)
	}
}
