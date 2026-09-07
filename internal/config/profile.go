// Package config resolves which archive the process is operating on.
//
// A profile is a data directory, not a domain concept. It is an
// organizational boundary with no security claim: the same OS user can
// read every profile's files regardless. Encoding it as a query-time
// column would advertise isolation the process cannot enforce.
package config

import (
	"os"
	"path/filepath"
	"strings"

	ierr "github.com/omkar273/burrow/internal/errors"
)

// EnvProfile is the environment variable that selects a profile.
const EnvProfile = "BURROW_PROFILE"

// DefaultProfile is used when neither flag nor environment selects one.
const DefaultProfile = "default"

// Profile is the resolved on-disk layout of a single archive.
type Profile struct {
	Name       string
	Root       string
	StateDB    string
	ObjectDir  string
	ConfigFile string
}

// ResolveProfile determines the active profile. Precedence is
// flag, then environment, then the default.
//
// lookupEnv and homeDir are parameters rather than package-level lookups
// so that tests need no global state.
func ResolveProfile(flagValue string, lookupEnv func(string) string, homeDir string) (Profile, error) {
	name := flagValue
	if name == "" {
		name = lookupEnv(EnvProfile)
	}
	if name == "" {
		name = DefaultProfile
	}

	if err := validateName(name); err != nil {
		return Profile{}, err
	}

	root := filepath.Join(homeDir, ".burrow", "profiles", name)
	return Profile{
		Name:       name,
		Root:       root,
		StateDB:    filepath.Join(root, "state.db"),
		ObjectDir:  filepath.Join(root, "objects"),
		ConfigFile: filepath.Join(root, "config.toml"),
	}, nil
}

// validateName rejects anything that could escape the profiles directory.
// A profile name becomes a path segment, so traversal here would let a
// flag value write anywhere on the filesystem.
func validateName(name string) error {
	switch {
	case name == "":
		return ierr.New("profile name is empty").
			WithHint("pass --profile <name> or set " + EnvProfile).
			Mark(ierr.ErrValidation)
	case name == "." || name == "..":
		return ierr.New("profile name is a path traversal: " + name).
			WithHint("use a plain name such as `work` or `personal`").
			Mark(ierr.ErrValidation)
	case strings.ContainsAny(name, `/\`):
		return ierr.New("profile name contains a path separator: " + name).
			WithHint("use a plain name such as `work` or `personal`").
			Mark(ierr.ErrValidation)
	}
	return nil
}

// Ensure creates the profile's directories if they do not exist. The mode
// is 0700 because this tree holds an archive of the user's mail.
func (p Profile) Ensure() error {
	for _, dir := range []string{p.Root, p.ObjectDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return ierr.Wrap(err, "creating profile directory "+dir).
				Mark(ierr.ErrInternal)
		}
		// MkdirAll respects umask, which can leave group or other bits
		// set; set the mode explicitly.
		if err := os.Chmod(dir, 0o700); err != nil {
			return ierr.Wrap(err, "setting mode on "+dir).
				Mark(ierr.ErrInternal)
		}
	}
	return nil
}
