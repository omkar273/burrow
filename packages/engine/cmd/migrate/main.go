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
			fmt.Fprintln(os.Stderr, "schema is up to date")
			return nil
		}
		// Say which case this is: against an existing archive the plan is a
		// delta, against no archive it is the whole schema. The SQL alone
		// does not distinguish them.
		version, err := engine.SchemaVersion(ctx, cfg)
		switch {
		case err != nil || version == 0:
			fmt.Fprintln(os.Stderr, "no archive yet — this is the full schema, not a delta")
		default:
			fmt.Fprintf(os.Stderr, "delta against the existing archive (schema version %d)\n", version)
		}
		// Only the SQL goes to stdout, so `migrate --dry-run > out.sql`
		// produces a file you can feed straight to sqlite3.
		fmt.Fprint(out, plan)
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
