package storage

import (
	"archive/tar"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/idtools"
	"github.com/containers/storage/pkg/ioutils"
	"github.com/containers/storage/pkg/reexec"
	"github.com/containers/storage/pkg/stringid"
)

var (
	topwd = ""
)

func TestMain(m *testing.M) {
	if reexec.Init() {
		return
	}
	wd, err := ioutil.TempDir("", "test.")
	if err != nil {
		os.Exit(1)
	}
	topwd = wd
	flag.Parse()
	code := m.Run()
	os.RemoveAll(wd)
	os.Exit(code)
}

func newStore(t *testing.T, driver string) Store {
	if driver == "" {
		driver = "vfs"
	}
	wd, err := ioutil.TempDir(topwd, "test.")
	if err != nil {
		t.Fatal(err)
	}
	err = os.MkdirAll(wd, 0700)
	if err != nil {
		t.Fatal(err)
	}
	run := filepath.Join(wd, "run")
	root := filepath.Join(wd, "root")
	uidmap := []idtools.IDMap{{
		ContainerID: 0,
		HostID:      os.Getuid(),
		Size:        1,
	}}
	gidmap := []idtools.IDMap{{
		ContainerID: 0,
		HostID:      os.Getgid(),
		Size:        1,
	}}
	store, err := GetStore(StoreOptions{
		RunRoot:            run,
		GraphRoot:          root,
		GraphDriverName:    driver,
		GraphDriverOptions: []string{},
		UidMap:             uidmap,
		GidMap:             gidmap,
	})
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func makeLayer(t *testing.T, compression archive.Compression) (string, int, int, []byte) {
	var cwriter io.WriteCloser
	var uncompressed *ioutils.WriteCounter
	var twriter *tar.Writer
	preader, pwriter := io.Pipe()
	tbuffer := bytes.Buffer{}
	if compression != archive.Uncompressed {
		compressor, err := archive.CompressStream(pwriter, compression)
		if err != nil {
			t.Fatalf("Error compressing layer: %v", err)
		}
		cwriter = compressor
		uncompressed = ioutils.NewWriteCounter(cwriter)
	} else {
		uncompressed = ioutils.NewWriteCounter(pwriter)
	}
	twriter = tar.NewWriter(uncompressed)
	buf := make([]byte, 12345)
	n, err := rand.Read(buf)
	if err != nil {
		t.Fatalf("Error reading tar data: %v", err)
	}
	if n != len(buf) {
		t.Fatalf("Short read reading tar data: %d < %d", n, len(buf))
	}
	for i := 1024; i < 2048; i++ {
		buf[i] = 0
	}
	go func() {
		defer pwriter.Close()
		if cwriter != nil {
			defer cwriter.Close()
		}
		defer twriter.Close()
		err := twriter.WriteHeader(&tar.Header{
			Name:       "/random-single-file",
			Mode:       0600,
			Size:       int64(len(buf)),
			ModTime:    time.Now(),
			AccessTime: time.Now(),
			ChangeTime: time.Now(),
			Typeflag:   tar.TypeReg,
		})
		if err != nil {
			t.Fatalf("Error writing tar header: %v", err)
		}
		n, err := twriter.Write(buf)
		if err != nil {
			t.Fatalf("Error writing tar header: %v", err)
		}
		if n != len(buf) {
			t.Fatalf("Short write writing tar header: %d < %d", n, len(buf))
		}
	}()
	_, err = io.Copy(&tbuffer, preader)
	if err != nil {
		t.Fatalf("Error reading layer tar: %v", err)
	}
	sum := sha256.Sum256(tbuffer.Bytes())
	return "sha256:" + hex.EncodeToString(sum[:]), int(uncompressed.Count), tbuffer.Len(), tbuffer.Bytes()
}

func TestGetStore(t *testing.T) {
	store1 := newStore(t, "")
	store2 := newStore(t, "")
	if store1 == store2 {
		t.Fatalf("GetStore: Got same pointer for two locations?")
	}
	store3, err := GetStore(StoreOptions{
		RunRoot:            store1.GetRunRoot(),
		GraphRoot:          store1.GetGraphRoot(),
		GraphDriverName:    store1.GetGraphDriverName(),
		GraphDriverOptions: store1.GetGraphOptions(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if store3 != store1 {
		t.Fatalf("GetStore: Got different pointer for single location (3).")
	}
	store4, err := GetStore(StoreOptions{
		GraphRoot:       store1.GetGraphRoot(),
		GraphDriverName: store1.GetGraphDriverName(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if store4 != store1 {
		t.Fatalf("GetStore: Got different pointer for single location (4).")
	}
	store5, err := GetStore(StoreOptions{
		GraphRoot: store1.GetGraphRoot(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if store5 != store1 {
		t.Fatalf("GetStore: Got different pointer for single location (5).")
	}
}

func TestGetLockfile(t *testing.T) {
	err := ioutils.AtomicWriteFile(filepath.Join(topwd, "TestGetLockfile"), []byte(stringid.GenerateRandomID()), 0600)
	if err != nil {
		t.Fatal(err)
	}
	lf, err := GetLockfile(filepath.Join(topwd, "TestGetLockfile"))
	if err != nil {
		t.Fatal(err)
	}
	lf.Lock()
	modified, err := lf.Modified()
	if err != nil {
		t.Fatal(err)
	}
	if !modified {
		t.Fatal("Modified should return true at startup")
	}
	then := time.Now().Add(-10 * time.Millisecond)
	lf.Touch()
	modified, err = lf.Modified()
	if err != nil {
		t.Fatal(err)
	}
	if modified {
		t.Fatal("Modified should be false if we touched it last")
	}
	if !lf.TouchedSince(then) {
		t.Fatal("Should have known we touched it since then")
	}
	if lf.TouchedSince(time.Now()) {
		t.Fatal("Haven't touched it yet")
	}
	lf.Unlock()
}

func TestCreateLayer(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("TestWriteRead requires root privileges")
	}

	layerNames := []string{"layer1", "layer2", "layer3"}
	store := newStore(t, "")

	parent := ""
	for _, layerName := range layerNames {
		layer, err := store.CreateLayer("", parent, []string{layerName}, "", true)
		if err != nil {
			t.Fatal(err)
		}
		mountPoint, err := store.Mount(layer.ID, "")
		if err != nil {
			t.Fatal(err)
		}
		mountPoint2, err := store.Mount(layer.ID, "")
		if err != nil {
			t.Fatal(err)
		}
		if mountPoint != mountPoint2 {
			t.Fatal("mount point moved")
		}

		contents := stringid.GenerateRandomID()
		err = ioutils.AtomicWriteFile(filepath.Join(mountPoint, layerName+"file"), []byte(contents), 0600)
		if err != nil {
			t.Fatal(err)
		}

		err = store.Unmount(layer.ID)
		if err != nil {
			t.Fatal(err)
		}
		err = store.Unmount(mountPoint)
		if err != nil {
			t.Fatal(err)
		}
		parent = layerName
	}

	for _, layerName := range layerNames {
		changes, err := store.Changes("", layerName)
		if err != nil {
			t.Fatal(err)
		}
		if len(changes) != 1 {
			t.Fatalf("expected 1 change in %q, got %d", layerName, len(changes))
		}
		if changes[0].Path != fmt.Sprintf("%c%s%s", filepath.Separator, layerName, "file") {
			t.Fatalf("expected %q, got %q", fmt.Sprintf("%c%s%s", filepath.Separator, layerName, "file"), changes[0].Path)
		}
		if changes[0].Kind != archive.ChangeAdd {
			t.Fatalf("expected ChangeAdd, got %v", changes[0].Kind)
		}
	}
}
