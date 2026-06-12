package main

import (
	"io"
	"os"
	"testing"
)

// TestMainWiring runs main with substitute arguments and checks that the
// command line, standard output and exit code are wired up correctly.
func TestMainWiring(t *testing.T) {
	origArgs, origStdout, origExit := os.Args, os.Stdout, exit
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	code := -1
	os.Args = []string{"pct", "ev", "20 + 10%"}
	os.Stdout = w
	exit = func(c int) { code = c }
	defer func() { os.Args, os.Stdout, exit = origArgs, origStdout, origExit }()

	main()

	w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Errorf("main() exited with %d, want 0", code)
	}
	if got, want := string(out), "22\n"; got != want {
		t.Errorf("main() printed %q, want %q", got, want)
	}
}
