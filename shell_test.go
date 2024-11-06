package main

import (
	"testing"
)

func TestFilterEnv(t *testing.T) {

	env := []string{
		"FOO=bar",
		"BAZ=1234",
		"AUDIT_UPLOAD_URL=http://foo",
	}
	res := filterEnv(env)

	if len(res) != 2 {
		t.Error("environments were not filtered")
	}
}
