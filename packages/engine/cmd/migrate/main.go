// Command migrate applies Burrow's versioned schema migrations, or prints
// what it would apply.
//
// Separate from burrow because migrating and serving are different
// operations with different blast radii: --dry-run must be able to show an
// operator the statements before anything touches a database holding their
// mail.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/omkar273/burrow/packages/engine"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func run(args []string, out *os.File) error {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	profile := fs.String("profile", "", "profile to migrate (default: $BURROW_PROFILE, then \"default\")")
	dryRun := fs.Bool("dry-run", false, "print the pending statements without applying them")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx := context.Background()
	cfg := engine.Config{Profile: *profile}

	if *dryRun {
		plan, err := engine.MigrationPlan(ctx, cfg)
		if err != nil {
			return err
		}
		if len(plan) == 0 {
			fmt.Fprintln(out, "no pending migrations")
			return nil
		}
		fmt.Fprintf(out, "%d pending migration(s), not applied:\n\n", len(plan))
		for _, m := range plan {
			fmt.Fprintf(out, "-- %s\n%s\n", m.Name, m.SQL)
		}
		return nil
	}

	applied, err := engine.Migrate(ctx, cfg)
	if err != nil {
		return err
	}
	if len(applied) == 0 {
		fmt.Fprintln(out, "no pending migrations")
		return nil
	}
	for _, name := range applied {
		fmt.Fprintf(out, "applied %s\n", name)
	}
	return nil
}
