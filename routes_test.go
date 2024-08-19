package main

import (
	"testing"
)

func TestBuildRoutesWithoutPrefix(t *testing.T) {
	r := BuildRoutes("")

	if r.Main != "/{$}" {
		t.Errorf("Main should be /{$}, got %s", r.Main)
	}

	if r.Shell != "/shell" {
		t.Errorf("Shell should be /shell, got %s", r.Shell)
	}

	if r.Home != "/home" {
		t.Errorf("Home should be /home, got %s", r.Home)
	}

	if r.Upload != "/upload" {
		t.Errorf("Upload should be /home, got %s", r.Upload)
	}

	if r.GetFile != "/home/{filename...}" {
		t.Errorf("GetFile should be /home/{filename...}, got %s", r.GetFile)
	}

	if r.Assets != "/assets/" {
		t.Errorf("Assets should be /assets/, got %s", r.Assets)
	}
}

func TestBuildRoutesWithPrefix(t *testing.T) {
	r := BuildRoutes("1234")

	if r.Main != "/1234/{$}" {
		t.Errorf("Main should be /1234/{$}, got %s", r.Main)
	}

	if r.Shell != "/1234/shell" {
		t.Errorf("Shell should be /1234/shell, got %s", r.Shell)
	}

	if r.Home != "/1234/home" {
		t.Errorf("Home should be /1234/home, got %s", r.Home)
	}

	if r.Upload != "/1234/upload" {
		t.Errorf("Home should be /upload, got %s", r.Upload)
	}

	if r.GetFile != "/1234/home/{filename...}" {
		t.Errorf("GetFile should be /1234/home/{filename...}, got %s", r.GetFile)
	}

	if r.Assets != "/1234/assets/" {
		t.Errorf("Assets should be /1234/assets/, got %s", r.Assets)
	}
}
