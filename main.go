// Constelate (Const.Elate) is a terminal character sheet and skill tree for
// Claude Code. It edits the plain files Claude Code already reads and never
// runs an agent of its own.
package main

import (
	"fmt"
	"os"

	"github.com/signalandform/constelate/cmd"
)

func main() {
	if err := cmd.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "constelate:", err)
		os.Exit(1)
	}
}
