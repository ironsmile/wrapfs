package wrapfs_test

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ironsmile/wrapfs"
)

//go:embed modtimefs.go
var testFS embed.FS

// ExampleWithModTime makes sure that an embed.FS will still support If-Modified-Since
// HTTP headers when used with http.FileServer.
func ExampleWithModTime() {
	modTime := time.Unix(1727600261, 0)

	modTimeFS := wrapfs.WithModTime(testFS, modTime)

	handler := http.FileServer(http.FS(modTimeFS))
	req := httptest.NewRequest(http.MethodGet, "/modtimefs.go", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)
	defer resp.Result().Body.Close()

	fmt.Printf("Last-Modified header: %s\n", resp.Result().Header.Get("Last-Modified"))
	// Output: Last-Modified header: Sun, 29 Sep 2024 08:57:41 GMT
}

// ExampleWithModTime_second makes sure that when If-Modified-Since is used then
// the file server returns 304.
func ExampleWithModTime_second() {
	modTime := time.Unix(1727600261, 0)
	modSince := modTime.Add(1 * time.Hour)

	modTimeFS := wrapfs.WithModTime(testFS, modTime)

	handler := http.FileServer(http.FS(modTimeFS))
	req := httptest.NewRequest(http.MethodGet, "/modtimefs.go", nil)
	req.Header.Set("If-Modified-Since", modSince.UTC().Format(http.TimeFormat))
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)
	defer resp.Result().Body.Close()

	fmt.Printf("HTTP Status Code: %d\n", resp.Result().StatusCode)
	// Output: HTTP Status Code: 304
}

// TestWithModTimeStat checks that the wrapped fs.FS implements fs.StatFS and also
// returns the expected mod time.
func TestWithModTimeStat(t *testing.T) {
	t.Parallel()

	modTime := time.Unix(1727600261, 0)
	modTimeFS := wrapfs.WithModTime(testFS, modTime)

	st, err := fs.Stat(modTimeFS, "modtimefs.go")
	if err != nil {
		t.Fatalf("fs.Stat returned an error: %s\n", err)
	}

	actualModTime := st.ModTime()
	if modTime != actualModTime {
		t.Errorf("expected mod time %s but got %s", modTime, actualModTime)
	}
}

// TestWithModTimeReadDir makes sure that using the fs.ReadDir preserves the used
// mod time for dir entries.
func TestWithModTimeReadDir(t *testing.T) {
	t.Parallel()

	modTime := time.Unix(1727600261, 0)
	modTimeFS := wrapfs.WithModTime(testFS, modTime)

	entries, err := fs.ReadDir(modTimeFS, ".")
	if err != nil {
		t.Fatalf("fs.ReadDir error: %s", err)
	}

	checkEntries(t, modTime, entries)
}

// TestWithModTimeOpenedFileStat checks that opened files return the set modification time
// when their Stat() method is called.
func TestWithModTimeOpenedFileStat(t *testing.T) {
	t.Parallel()

	modTime := time.Unix(1727600261, 0)
	modTimeFS := wrapfs.WithModTime(testFS, modTime)

	fh, err := modTimeFS.Open("modtimefs.go")
	if err != nil {
		t.Fatalf("fs.Open returned an error: %s\n", err)
	}
	defer fh.Close()

	st, err := fh.Stat()
	if err != nil {
		t.Fatalf("File.Stat returned an error: %s\n", err)
	}

	actualModTime := st.ModTime()
	if modTime != actualModTime {
		t.Errorf("expected mod time %s but got %s", modTime, actualModTime)
	}
}

// TestWithModTimeOpenedDirReadDir checks that opened directories which implemented
// ReadDir also return the mod time for their entries.
func TestWithModTimeOpenedDirReadDir(t *testing.T) {
	t.Parallel()

	modTime := time.Unix(1727600261, 0)
	modTimeFS := wrapfs.WithModTime(testFS, modTime)

	fh, err := modTimeFS.Open(".")
	if err != nil {
		t.Fatalf("fs.Open returned an error: %s\n", err)
	}
	defer fh.Close()

	rd, ok := fh.(readerDir)
	if !ok {
		t.Fatalf("opened dir is not a readerDir")
	}

	entries, err := rd.ReadDir(10)
	if err != nil {
		t.Fatalf("Dir.ReadDir returned an error: %s\n", err)
	}

	checkEntries(t, modTime, entries)
}

func checkEntries(t *testing.T, modTime time.Time, entries []fs.DirEntry) {
	for _, entry := range entries {
		st, err := entry.Info()
		if err != nil {
			t.Fatalf("[%s] entry.Info returned an error: %s\n", entry.Name(), err)
		}

		actualModTime := st.ModTime()
		if modTime != actualModTime {
			t.Errorf("[%s] expected mod time %s but got %s",
				entry.Name(), modTime, actualModTime,
			)
		}
	}
}

type readerDir interface {
	ReadDir(count int) ([]fs.DirEntry, error)
}
