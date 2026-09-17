package gitidentity

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Profile struct{ Name, File string }

var Profiles = []Profile{{"GitHub personal", "github_personal.gitconfig"}, {"Work", "work.gitconfig"}, {"GitLab", "gitlab.gitconfig"}}

func Path(home string, profile Profile) string {
	return filepath.Join(home, ".config", "git", "identities", profile.File)
}

func Configured(home string, profile Profile) bool {
	for _, key := range []string{"user.name", "user.email"} {
		output, err := exec.Command("git", "config", "--file", Path(home, profile), "--get", key).Output()
		if err != nil || strings.TrimSpace(string(output)) == "" {
			return false
		}
	}
	return true
}

// Create never replaces a user's existing identity file.
func Create(home string, profile Profile, name, email string) error {
	name, email = strings.TrimSpace(name), strings.TrimSpace(email)
	if name == "" || email == "" || strings.ContainsAny(name+email, "\r\n\x00") {
		return errors.New("Git identity needs a name and email on single lines")
	}
	dir := home
	for _, part := range []string{".config", "git", "identities"} {
		dir = filepath.Join(dir, part)
		info, err := os.Lstat(dir)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(dir, 0o700); err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else if !info.IsDir() {
			return fmt.Errorf("%s must be a real directory", dir)
		}
	}
	file, err := os.OpenFile(Path(home, profile), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, writeErr := fmt.Fprintf(file, "[user]\n    name = %s\n    email = %s\n", strconv.Quote(name), strconv.Quote(email))
	return errors.Join(writeErr, file.Close())
}
