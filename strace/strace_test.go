package strace

import (
	"testing"
)

var valid = []string{
	`[pid 3702590] 1730503723.010946 execve("/usr/bin/whoami", ["whoami"], 0x5601036f6790 /* 52 vars */) = 0`,
	`[pid 306753] 1730709070.467006 execve("/usr/bin/ls", ["ls", "--color=auto", "/home/chris/code/go/", "-adfg", "-g"], 0x557bf130cb90 /* 53 vars */) = 0`,
	`[pid    17] 1730823571.031830 execve("/usr/bin/ls", ["ls", "-la"], 0x557237896740 /* 7 vars */) = 0`,
}

var invalid = []string{
	`[pid 306705] 1730709057.951196 +++ exited with 0 +++`,
	`1730709057.952107 --- SIGCHLD {si_signo=SIGCHLD, si_code=CLD_EXITED, si_pid=306705, si_uid=1000, si_status=0, si_utime=0, si_stime=0} ---`,
	`strace: Process 306753 attached`,
	`[pid 306753] 1730709070.472990 +++ exited with 0 +++`,
	`1730709070.473327 --- SIGCHLD {si_signo=SIGCHLD, si_code=CLD_EXITED, si_pid=306753, si_uid=1000, si_status=0, si_utime=0, si_stime=0} ---`,
}

func TestFilter(t *testing.T) {
	for _, s := range valid {
		if !filter(s) {
			t.Errorf("Filter failed rejected %s", s)
		}
	}

	for _, s := range invalid {
		if filter(s) {
			t.Errorf("Filter failed accepted %s", s)
		}
	}
}

func TestParse(t *testing.T) {
	res, err := parse(`[pid 3702590] 1730503723.010946 execve("/usr/bin/whoami", ["whoami"], 0x5601036f6790 /* 52 vars */) = 0`)
	if err != nil {
		t.Fatal(err)
	}

	if res.Pid != "3702590" {
		t.Errorf("invalid pid want 3702590 got %s", res.Pid)
	}

	if res.Cmd != `"/usr/bin/whoami", ["whoami"], 0x5601036f6790 /* 52 vars */` {
		t.Errorf("invalid pid want 3702590 got %s", res.Pid)
	}
}

func TestParseMany(t *testing.T) {
	for _, s := range valid {
		_, err := parse(s)
		if err != nil {
			t.Errorf("Failed to parse: %s", s)
		}

	}
}
