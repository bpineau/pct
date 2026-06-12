// Pct is a terminal calculator that speaks percentages.
//
// It evaluates arithmetic expressions in which percentages adapt to their
// context ("pct eval '20 + 10%'" prints 22) and offers fixed-form commands
// for the everyday questions: adding a percentage to a number, finding the
// percentage change between two numbers, the share one number represents
// of another, and compound growth. Run without arguments it evaluates
// expressions from standard input, interactively when attached to a
// terminal. Run "pct help" for details.
package main

import (
	"os"

	"github.com/bpineau/pct/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
