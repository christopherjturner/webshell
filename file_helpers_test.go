package main

import (
	"testing"
)

func TestIsPathSafe(t *testing.T) {

	config.HomeDir = "/home/foo"

	paths := []string{
		"/home/foo",
		"/home/foo/",
		"/home/foo/bar",
		"/home/foo/.bar/baz",
	}

	for _, p := range paths {
		if !isPathSafe(p) {
			t.Errorf("expected path to be safe %s", p)
		}
	}
}

func TestIsPathUnsafe(t *testing.T) {

	config.HomeDir = "/home/foo"

	paths := []string{
		"/home/foo/../../../root",
		"/usr/local/bin",
		"../home/foo/bar",
		"/test",
	}

	for _, p := range paths {
		if isPathSafe(p) {
			t.Errorf("expected path to be unsafe %s", p)
		}
	}
}
