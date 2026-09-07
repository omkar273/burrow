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
		if plan == "" {
			fmt.Fprintln(out, "schema is up to date")
			return nil
		}
		fmt.Fprintln(out, "pending statements, not applied:")
		fmt.Fprintln(out)
		fmt.Fprintln(out, plan)
		return nil
	}

	before, err := engine.SchemaVersion(ctx, cfg)
	if err != nil {
		return err
	}
	if err := engine.Migrate(ctx, cfg); err != nil {
		return err
	}
	after, err := engine.SchemaVersion(ctx, cfg)
	if err != nil {
		return err
	}
	if before == after {
		fmt.Fprintf(out, "schema is up to date (version %d)\n", after)
		return nil
	}
	fmt.Fprintf(out, "migrated: schema version %d -> %d\n", before, after)
	return nil
}
