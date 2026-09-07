// Command burrowd is the local Burrow daemon and CLI.
//
// It is a thin consumer of packages/engine: everything it can reach is
// the engine's public surface, which is what keeps the engine's internals
// free to change. Commands land in Task 13 of the M0/M1 plan.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/omkar273/burrow/packages/engine"
)

func main() {
	if err := run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "burrowd:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	profile := ""
	if len(args) > 2 && args[1] == "--profile" {
		profile = args[2]
	}

	e, err := engine.Open(context.Background(), engine.Config{Profile: profile})
	if err != nil {
		return err
	}
	defer e.Close()

	p := e.Paths()
	fmt.Printf("profile   %s\n", p.Profile)
	fmt.Printf("state     %s\n", p.StateDB)
	fmt.Printf("objects   %s\n", p.ObjectDir)
	return nil
}
