package setup

import (
	"errors"
	"fmt"
	"os"

	"github.com/emmsixx/dotfiles/internal/gitidentity"
	"github.com/emmsixx/dotfiles/internal/ui"
)

func (r Runner) configureGitIdentities(o Options) error {
	for _, profile := range gitidentity.Profiles {
		if gitidentity.Configured(o.Home, profile) {
			continue
		}
		path := gitidentity.Path(o.Home, profile)
		if _, err := os.Lstat(path); err == nil {
			ui.Warn("complete user.name and user.email in " + path)
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if !o.canPrompt() {
			ui.Warn("configure the " + profile.Name + " Git identity in " + path)
			continue
		}
		configure := false
		if err := ui.Confirm("Configure "+profile.Name+" Git identity?", "Repos in this profile need an explicit commit name and email.", &configure); err != nil {
			return err
		}
		if !configure {
			continue
		}
		var name, email string
		if err := ui.Inputs(profile.Name+" Git identity", []ui.Input{{Title: "Commit author name", Value: &name}, {Title: "Commit author email", Value: &email}}); err != nil {
			return err
		}
		if err := gitidentity.Create(o.Home, profile, name, email); err != nil {
			return fmt.Errorf("configure %s: %w", profile.Name, err)
		}
	}
	return nil
}
