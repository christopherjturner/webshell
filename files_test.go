package main

import (
	"bytes"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setup() (*FilesHandler, error) {
	dir, err := os.MkdirTemp(os.TempDir(), "webshell-fs-test")
	if err != nil {
		return nil, err
	}

	if err = os.WriteFile(filepath.Join(dir, "foo.txt"), []byte("foo"), 0666); err != nil {
		return nil, err
	}

	if err = os.Mkdir(filepath.Join(dir, "bar"), 0755); err != nil {
		return nil, err
	}

	if err = os.WriteFile(filepath.Join(dir, "bar/baz.txt"), []byte("baz"), 0666); err != nil {
		return nil, err
	}

	h := &FilesHandler{
		baseDir: dir,
		baseUrl: "/1234/home",
		logger:  slog.Default(),
	}
	return h, nil
}

func teardown(f *FilesHandler) {
	defer os.RemoveAll(f.baseDir)
}

func TestGet(t *testing.T) {
	h, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer teardown(h)

	req := httptest.NewRequest(http.MethodGet, "/1234/home/foo.txt", nil)
	req.SetPathValue("filename", "foo.txt")

	w := httptest.NewRecorder()

	h.Handler().ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()
	if w.Code != 200 {
		t.Errorf("status code: want 200 got %d", w.Code)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	if string(data) != "foo" {
		t.Errorf("expected foo got %v", string(data))
	}
}

// Check we get the correct error code when trying to download a missing file.
func TestGet404(t *testing.T) {
	h, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer teardown(h)

	req := httptest.NewRequest(http.MethodGet, "/1234/home/missing-file", nil)
	req.SetPathValue("filename", "missing-file")

	w := httptest.NewRecorder()

	h.Handler().ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()
	if w.Code != 404 {
		t.Errorf("status code: want 404 got %d", w.Code)
	}
}

// Check home dir is listed correctly. It should contain:
// - foo.txt
// - no link to parent dir (this only shows on subdirs)
func TestListBase(t *testing.T) {
	h, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer teardown(h)

	req := httptest.NewRequest(http.MethodGet, "/1234/home/", nil)
	req.SetPathValue("filename", "")

	w := httptest.NewRecorder()

	h.Handler().ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()
	if w.Code != 200 {
		t.Errorf("status code: want 200 got %d", w.Code)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}

	fooLink := "<a href=\"/1234/home/foo.txt\" class=\"file\" target=\"_blank\">foo.txt</a>"
	parentLink := "<a href=\"/1234/home\" class=\"dir\">../</a>"

	if !strings.Contains(string(data), fooLink) {
		t.Errorf("expected %s got %s", fooLink, string(data))
	}

	if strings.Contains(string(data), parentLink) {
		t.Errorf("expected %s got %s", parentLink, string(data))
	}
}

// Check home dir is listed correctly. It should contain:
// - foo.txt
// - no link to parent dir (this only shows on subdirs)
func TestListSubDir(t *testing.T) {
	h, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer teardown(h)

	req := httptest.NewRequest(http.MethodGet, "/1234/home/bar", nil)
	req.SetPathValue("filename", "bar")

	w := httptest.NewRecorder()

	h.Handler().ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()
	if w.Code != 200 {
		t.Errorf("status code: want 200 got %d", w.Code)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}

	fooLink := "<a href=\"/1234/home/bar/baz.txt\" class=\"file\" target=\"_blank\">baz.txt</a>"
	parentLink := "<a href=\"/1234/home\" class=\"dir\">../</a>"

	if !strings.Contains(string(data), fooLink) {
		t.Errorf("expected %s got %s", fooLink, string(data))
	}

	if !strings.Contains(string(data), parentLink) {
		t.Errorf("expected %s got %s", parentLink, string(data))
	}
}

func TestUpload(t *testing.T) {

	var FILENAME string = "hello.txt"
	var FILE_CONTENT string = "hello world"

	h, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer teardown(h)

	// build multipart request
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", FILENAME)
	if err != nil {
		t.Fatalf("could not create form file: %v", err)
	}

	fileContent := bytes.NewBufferString(FILE_CONTENT)
	_, err = io.Copy(part, fileContent)
	if err != nil {
		t.Fatalf("could not copy file content: %v", err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatalf("could not close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/1234/home", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// make request
	w := httptest.NewRecorder()
	h.Handler().ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()
	if w.Code != http.StatusSeeOther {
		t.Errorf("status code: want %d got %d", http.StatusSeeOther, w.Code)
	}

	// Check file exists.
	uploadedFile, err := os.Open(filepath.Join(h.baseDir, FILENAME))
	if err != nil {
		t.Fatalf("Upload %s was not in dir %s", FILENAME, filepath.Join(h.baseDir, FILENAME))
	}

	// Check file has the right content.
	uploadedContent, err := io.ReadAll(uploadedFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(uploadedContent) != FILE_CONTENT {
		t.Errorf("Uploaed file was wrong, want: %s got: %s", FILE_CONTENT, uploadedContent)
	}

}

func TestAsLink(t *testing.T) {
	fh := FilesHandler{
		baseDir: "/home/user/",
		baseUrl: "/token/home",
		logger:  slog.Default(),
	}

	testCases := []struct {
		in  string
		out string
	}{
		{"/home/user/foo.txt", "/token/home/foo.txt"},
		{"/home/user/sub/foo.txt", "/token/home/sub/foo.txt"},
		{"/home/user", "/token/home"},
		{"/home/user/", "/token/home"},
		{"/home/user/sub/dir", "/token/home/sub/dir"},
		{"/home/user/sub/dir/", "/token/home/sub/dir"},
	}

	for _, test := range testCases {
		if fh.asLink(test.in) != test.out {
			t.Errorf("want %s got %s", test.in, test.out)
		}
	}
}
