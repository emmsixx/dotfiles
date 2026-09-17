package setup

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/emmsixx/dotfiles/internal/ui"
)

type linkSpec struct{ relative, source string }
type linkJournal struct{ undo []func() error }

func (j *linkJournal) rollback() error {
	var failures []error
	for i := len(j.undo) - 1; i >= 0; i-- {
		failures = append(failures, j.undo[i]())
	}
	return errors.Join(failures...)
}

// Link gives Stow the same tracked-file inventory used by the preflight. All
// home mutations, including folded directories and generated skills, are
// journaled; a failed operation restores the previous configuration.
func Link(repo, home string, run func(string, ...string) error) (result error) {
	var err error
	repo, err = filepath.EvalSymlinks(repo)
	if err != nil {
		return err
	}
	repo, err = filepath.Abs(repo)
	if err != nil {
		return err
	}
	home, err = filepath.EvalSymlinks(home)
	if err != nil {
		return err
	}
	home, err = filepath.Abs(home)
	if err != nil {
		return err
	}
	if err := ensureDistinctRepoAndHome(repo, home); err != nil {
		return err
	}
	tracked, err := managedFiles(repo)
	if err != nil {
		return err
	}
	generated, err := generatedSkills(repo, home)
	if err != nil {
		return err
	}
	specs := append(append([]linkSpec{}, tracked...), generated...)
	dirs := linkDirectories(specs)
	folded, err := preflightLinks(repo, home, dirs, specs)
	if err != nil {
		return err
	}

	backupParent := filepath.Join(home, ".dotfiles-backups")
	if info, err := os.Lstat(backupParent); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("backup directory %s must be a real directory", backupParent)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(backupParent, 0o700); err != nil {
		return err
	}
	backup, err := os.MkdirTemp(backupParent, "link-")
	if err != nil {
		return err
	}

	journal := &linkJournal{}
	defer func() {
		if result != nil {
			if err := journal.rollback(); err != nil {
				result = errors.Join(result, fmt.Errorf("restore previous configuration: %w", err))
			}
		}
	}()
	managed := map[string]bool{}
	for _, spec := range tracked {
		managed[spec.relative] = true
	}
	for _, dir := range dirs {
		target := filepath.Join(home, dir)
		if folded[dir] {
			staged := filepath.Join(backup, "unfolded", dir)
			if err := copyUnmanaged(filepath.Join(repo, dir), staged, dir, managed); err != nil {
				return err
			}
			saved := filepath.Join(backup, "links", dir)
			if err := os.MkdirAll(filepath.Dir(saved), 0o700); err != nil {
				return err
			}
			if err := os.Rename(target, saved); err != nil {
				return err
			}
			journal.undo = append(journal.undo, func() error {
				if _, err := os.Lstat(target); err == nil {
					// Keep the replacement tree as recovery data, including
					// anything an application wrote during the operation.
					if err := os.Rename(target, staged); err != nil {
						return err
					}
				} else if !errors.Is(err, os.ErrNotExist) {
					return err
				}
				return os.Rename(saved, target)
			})
			if err := os.Rename(staged, target); err != nil {
				return err
			}
		} else if _, err := os.Lstat(target); errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(target, 0o700); err != nil {
				return err
			}
			journal.undo = append(journal.undo, func() error { return os.Remove(target) })
		} else if err != nil {
			return err
		}
	}
	for _, spec := range specs {
		target, source := filepath.Join(home, spec.relative), spec.source
		info, err := os.Lstat(target)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err == nil {
			saved := filepath.Join(backup, "conflicts", spec.relative)
			if err := os.MkdirAll(filepath.Dir(saved), 0o700); err != nil {
				return err
			}
			if !info.Mode().IsRegular() && info.Mode()&fs.ModeSymlink == 0 {
				return fmt.Errorf("unexpected conflict at %s", target)
			}
			if err := os.Rename(target, saved); err != nil {
				return err
			}
			journal.undo = append(journal.undo, func() error {
				if err := removeInstalledLink(target, source); err != nil {
					return err
				}
				return os.Rename(saved, target)
			})
		} else {
			journal.undo = append(journal.undo, func() error { return removeInstalledLink(target, source) })
		}
	}
	allowed := map[string]bool{}
	for _, spec := range tracked {
		allowed[spec.relative] = true
		for dir := filepath.Dir(spec.relative); dir != "."; dir = filepath.Dir(dir) {
			allowed[dir] = true
		}
	}
	patterns := make([]string, 0, len(allowed))
	for path := range allowed {
		patterns = append(patterns, regexp.QuoteMeta(filepath.ToSlash(path)))
	}
	sort.Strings(patterns)
	// Stow uses Perl regexes on paths relative to the target. A negative
	// lookahead excludes every path outside the inventory, including new
	// runtime files that appear after preflight.
	ignore := "^(?!(?:" + strings.Join(patterns, "|") + ")$).*"
	if err := run("stow", "--no-folding", "--restow", "--ignore="+ignore, "-d", repo, "-t", home, "."); err != nil {
		return fmt.Errorf("stow dotfiles: %w", err)
	}
	for _, spec := range tracked {
		if !sameLinkTarget(filepath.Join(home, spec.relative), spec.source) {
			return fmt.Errorf("Stow did not link %s; check .stow-local-ignore", spec.relative)
		}
	}
	for _, spec := range generated {
		if err := os.Symlink(spec.source, filepath.Join(home, spec.relative)); err != nil {
			return err
		}
	}
	ui.OK("linked tracked dotfiles; recovery data: " + backup)
	return nil
}

func ensureDistinctRepoAndHome(repo, home string) error {
	repoInfo, err := os.Stat(repo)
	if err != nil {
		return err
	}
	homeInfo, err := os.Stat(home)
	if err != nil {
		return err
	}
	if os.SameFile(repoInfo, homeInfo) {
		return fmt.Errorf("refusing to link: repo %s and home %s are the same directory", repo, home)
	}
	relative, err := filepath.Rel(repo, home)
	if err != nil {
		return err
	}
	if relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("the home directory must not be inside the dotfiles repository")
	}
	return nil
}

func managedFiles(repo string) ([]linkSpec, error) {
	output, err := exec.Command("git", "-C", repo, "ls-files", "--cached", "--recurse-submodules", "-z").Output()
	if err != nil {
		return nil, fmt.Errorf("read tracked dotfiles (a Git checkout with initialized submodules is required): %w", err)
	}
	var files []linkSpec
	for _, relative := range strings.Split(string(output), "\x00") {
		if relative == "" || skipStowPath(relative) {
			continue
		}
		source := filepath.Join(repo, relative)
		info, err := os.Lstat(source)
		if errors.Is(err, os.ErrNotExist) {
			continue
		} // respect local deletions
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			return nil, fmt.Errorf("initialize submodule %s before linking", relative)
		}
		if !info.Mode().IsRegular() && info.Mode()&fs.ModeSymlink == 0 {
			return nil, fmt.Errorf("unsupported managed file %s", source)
		}
		for dir := filepath.Dir(source); dir != repo; dir = filepath.Dir(dir) {
			parent, err := os.Lstat(dir)
			if err != nil {
				return nil, err
			}
			if !parent.IsDir() {
				return nil, fmt.Errorf("managed source directory %s must not be a symlink", dir)
			}
		}
		files = append(files, linkSpec{relative, source})
	}
	if len(files) == 0 {
		return nil, errors.New("no tracked configuration files found")
	}
	return files, nil
}

func linkDirectories(specs []linkSpec) []string {
	unique := map[string]bool{".config": true}
	for _, spec := range specs {
		for dir := filepath.Dir(spec.relative); dir != "."; dir = filepath.Dir(dir) {
			unique[dir] = true
		}
	}
	dirs := make([]string, 0, len(unique))
	for dir := range unique {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs) // parents sort before their children
	return dirs
}

func underFolded(relative string, folded map[string]bool) bool {
	for dir := filepath.Dir(relative); dir != "."; dir = filepath.Dir(dir) {
		if folded[dir] {
			return true
		}
	}
	return false
}

func preflightLinks(repo, home string, dirs []string, specs []linkSpec) (map[string]bool, error) {
	folded := map[string]bool{}
	for _, relative := range dirs {
		if underFolded(relative, folded) {
			continue
		}
		target := filepath.Join(home, relative)
		info, err := os.Lstat(target)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if info.Mode()&fs.ModeSymlink != 0 && sameLinkTarget(target, filepath.Join(repo, relative)) {
			folded[relative] = true
		} else if !info.IsDir() {
			return nil, fmt.Errorf("preserving unrelated directory target %s; move it manually before linking", target)
		}
	}
	for _, spec := range specs {
		if underFolded(spec.relative, folded) {
			continue
		}
		target := filepath.Join(home, spec.relative)
		info, err := os.Lstat(target)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			if !sameLinkTarget(target, spec.source) {
				return nil, fmt.Errorf("preserving unrelated link %s", target)
			}
		} else if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("preserving non-file conflict %s", target)
		}
	}
	return folded, nil
}

func sameLinkTarget(link, source string) bool {
	destination, err := os.Readlink(link)
	if err != nil {
		return false
	}
	if !filepath.IsAbs(destination) {
		destination = filepath.Join(filepath.Dir(link), destination)
	}
	if filepath.Clean(destination) == filepath.Clean(source) {
		return true
	}
	left, leftErr := os.Stat(destination)
	right, rightErr := os.Stat(source)
	return leftErr == nil && rightErr == nil && os.SameFile(left, right)
}

func removeInstalledLink(target, source string) error {
	info, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&fs.ModeSymlink == 0 || !sameLinkTarget(target, source) {
		return fmt.Errorf("cannot roll back %s: it changed during linking; recovery data was retained", target)
	}
	return os.Remove(target)
}

func skipStowPath(relative string) bool {
	parts := strings.Split(filepath.ToSlash(relative), "/")
	for _, part := range parts {
		if part == ".git" || part == ".gitignore" || part == ".DS_Store" || part == "node_modules" {
			return true
		}
	}
	switch parts[0] {
	case ".agents", ".github", "cmd", "internal", "skills", ".firecrawl", ".secrets", ".dotfiles-backups":
		return true
	}
	path := filepath.ToSlash(relative)
	if strings.HasPrefix(parts[0], "LICENSE") {
		return true
	}
	if strings.HasPrefix(path, ".pi/agent/extensions/") {
		for _, part := range parts {
			if part == "bin" {
				return true
			}
		}
	}
	if strings.HasPrefix(path, ".codex/") && path != ".codex/AGENTS.md" && !strings.HasPrefix(path, ".codex/rules/") {
		return true
	}
	if strings.HasPrefix(path, ".claude/") && path != ".claude/CLAUDE.md" && path != ".claude/settings.json" {
		return true
	}
	if strings.HasPrefix(path, ".pi/") && path != ".pi/agent/AGENTS.md" && !strings.HasPrefix(path, ".pi/agent/extensions/") && !strings.HasPrefix(path, ".pi/agent/skills/") {
		return true
	}
	switch path {
	case ".config/nvim/colors/dms.lua", ".config/nvim/lua/plugins/dankcolors.lua",
		".config/ghostty/themes/dankcolors", ".config/niri/colors.kdl":
		return true
	}
	for _, prefix := range []string{".config/gh", ".config/firecrawl-cli", ".config/astro", ".config/systemd", ".config/git/identities", ".config/niri/dms"} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	if strings.Contains(path, "/config.private.json") || strings.HasPrefix(path, ".pi/agent/extensions/bin/") {
		return true
	}
	switch relative {
	case "README.md", "AGENTS.md", "LICENSE", "setup.sh", "setup-server.sh", ".secrets.example", ".gitmodules", ".skills", ".node-version", "skills-lock.json", ".stow-local-ignore", "go.mod", "go.sum", ".goreleaser.yaml", "dotfiles":
		return true
	}
	return false
}

func generatedSkills(repo, home string) ([]linkSpec, error) {
	var result []linkSpec
	for _, location := range [][2]string{{".claude/skills", ".claude/skills"}, {".pi/skills", ".pi/agent/skills"}} {
		entries, err := os.ReadDir(filepath.Join(repo, location[0]))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			source := filepath.Join(repo, location[0], entry.Name())
			relative := filepath.Join(location[1], entry.Name())
			target := filepath.Join(home, relative)
			if _, err := os.Stat(source); err != nil {
				return nil, fmt.Errorf("read generated skill %s: %w", source, err)
			}
			if _, err := os.Lstat(target); err == nil && !sameLinkTarget(target, source) {
				ui.Warn("preserving existing generated-skill target " + target)
				continue
			} else if err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
			result = append(result, linkSpec{relative, source})
		}
	}
	return result, nil
}

// Copy everything except managed leaves before replacing a folded directory.
// Untracked state inside managed application directories is preserved too.
func copyUnmanaged(source, destination, relative string, managed map[string]bool) error {
	if managed[relative] {
		return nil
	}
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return copyTree(source, destination)
	}
	if err := os.MkdirAll(destination, info.Mode().Perm()); err != nil {
		return err
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() == ".git" {
			continue
		}
		if err := copyUnmanaged(filepath.Join(source, entry.Name()), filepath.Join(destination, entry.Name()), filepath.Join(relative, entry.Name()), managed); err != nil {
			return err
		}
	}
	return nil
}

func copyTree(source, destination string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		link, err := os.Readlink(source)
		if err != nil {
			return err
		}
		if !filepath.IsAbs(link) {
			link = filepath.Join(filepath.Dir(source), link)
		}
		return os.Symlink(link, destination)
	}
	if info.Mode().IsRegular() {
		return copyFile(source, destination, info.Mode())
	}
	if !info.IsDir() {
		return fmt.Errorf("cannot safely copy runtime file %s (%s)", source, info.Mode())
	}
	if err := os.MkdirAll(destination, info.Mode().Perm()); err != nil {
		return err
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := copyTree(filepath.Join(source, entry.Name()), filepath.Join(destination, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(source, destination string, mode fs.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return err
	}
	out, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode.Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	return errors.Join(copyErr, out.Close())
}
