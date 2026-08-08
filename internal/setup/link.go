package setup

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emmsixx/dotfiles/internal/ui"
)

// Link uses Stow without directory folding. In particular, ~/.config is kept
// as a normal directory so applications can write their own untracked state.
func Link(repo, home string, run func(name string, args ...string) error) error {
	if err := ensureDistinctRepoAndHome(repo, home); err != nil {
		return err
	}
	if err := unfoldConfig(repo, home); err != nil {
		return err
	}
	if err := removeAbsoluteManagedLinks(repo, home); err != nil {
		return err
	}
	restore, err := backUpStowConflicts(repo, home)
	if err != nil {
		return err
	}
	if err := run("stow", "--no-folding", "--restow", "-d", repo, "-t", home, "."); err != nil {
		restore()
		return fmt.Errorf("stow dotfiles: %w", err)
	}
	if err := linkGeneratedSkills(repo, home); err != nil {
		return err
	}
	ui.OK("linked tracked dotfiles without folding ~/.config")
	return nil
}

// ensureDistinctRepoAndHome guards against repo and home resolving to the
// same directory. backUpStowConflicts and removeAbsoluteManagedLinks rename
// and remove paths under home on the assumption that home is never the repo
// itself; if they alias, those "conflict" cleanups delete the repo's only
// copy of a tracked file instead of a stray copy under home.
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
	return nil
}

// removeAbsoluteManagedLinks removes only links that point directly at the
// current checkout's managed path. Stow deliberately refuses those absolute
// links because they are not portable; removing them lets --restow recreate a
// normal relative link. Unrelated links are left untouched.
func removeAbsoluteManagedLinks(repo, home string) error {
	return filepath.WalkDir(repo, func(source string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(repo, source)
		if err != nil {
			return err
		}
		if relative == "." || skipStowPath(relative) {
			if entry.IsDir() && relative != "." {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(home, relative)
		info, err := os.Lstat(target)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&fs.ModeSymlink == 0 {
			return nil
		}
		link, err := os.Readlink(target)
		if err != nil {
			return err
		}
		if !filepath.IsAbs(link) || filepath.Clean(link) != filepath.Clean(source) {
			return nil
		}
		if err := os.Remove(target); err != nil {
			return fmt.Errorf("remove absolute managed link %s: %w", target, err)
		}
		ui.Warn("replaced non-portable absolute link " + target)
		return nil
	})
}

type movedTarget struct {
	original string
	backup   string
}

// backUpStowConflicts moves only regular source-file conflicts, never whole
// configuration directories. This keeps provider-owned state in ~/.claude,
// ~/.config, and similar roots intact while preserving user-edited files.
func backUpStowConflicts(repo, home string) (func(), error) {
	backupRoot := filepath.Join(home, ".dotfiles-backups", time.Now().Format("20060102-150405"), "conflicts")
	moved := []movedTarget{}
	err := filepath.WalkDir(repo, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(repo, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		if entry.IsDir() {
			if skipStowPath(relative) {
				return filepath.SkipDir
			}
			return nil
		}
		if skipStowPath(relative) {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return err
		}
		target := filepath.Join(home, relative)
		targetInfo, err := os.Lstat(target)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if targetInfo.Mode()&fs.ModeSymlink != 0 {
			return nil
		}
		backup := filepath.Join(backupRoot, relative)
		if err := os.MkdirAll(filepath.Dir(backup), 0o700); err != nil {
			return err
		}
		if err := os.Rename(target, backup); err != nil {
			return fmt.Errorf("back up %s: %w", target, err)
		}
		moved = append(moved, movedTarget{original: target, backup: backup})
		ui.Warn("moved existing " + target + " to " + backup)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return func() {
		for index := len(moved) - 1; index >= 0; index-- {
			item := moved[index]
			if info, err := os.Lstat(item.original); err == nil && info.Mode()&fs.ModeSymlink != 0 {
				_ = os.Remove(item.original)
			}
			if _, err := os.Lstat(item.original); errors.Is(err, os.ErrNotExist) {
				_ = os.MkdirAll(filepath.Dir(item.original), 0o700)
				_ = os.Rename(item.backup, item.original)
			}
		}
	}, nil
}

func skipStowPath(relative string) bool {
	first := strings.Split(relative, string(filepath.Separator))[0]
	if first == ".git" || first == ".agents" || first == ".github" || first == "cmd" || first == "internal" || first == "skills" {
		return true
	}
	if strings.HasPrefix(relative, filepath.Join(".claude", "skills")) || strings.HasPrefix(relative, filepath.Join(".pi", "skills")) {
		return true
	}
	switch relative {
	case "README.md", "LICENSE", "setup.sh", "setup-server.sh", ".secrets.example", ".gitmodules", ".skills", ".node-version", "skills-lock.json", ".stow-local-ignore", "go.mod", "go.sum", ".goreleaser.yaml":
		return true
	}
	return false
}

func linkGeneratedSkills(repo, home string) error {
	locations := []struct {
		source string
		target string
	}{
		{source: filepath.Join(".claude", "skills"), target: filepath.Join(".claude", "skills")},
		{source: filepath.Join(".pi", "skills"), target: filepath.Join(".pi", "agent", "skills")},
	}
	for _, location := range locations {
		source := filepath.Join(repo, location.source)
		entries, err := os.ReadDir(source)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		targetDirectory := filepath.Join(home, location.target)
		if err := os.MkdirAll(targetDirectory, 0o700); err != nil {
			return err
		}
		for _, entry := range entries {
			sourcePath := filepath.Join(source, entry.Name())
			targetPath := filepath.Join(targetDirectory, entry.Name())
			if existing, err := os.Lstat(targetPath); err == nil {
				if existing.Mode()&fs.ModeSymlink == 0 {
					ui.Warn("preserving existing generated-skill target " + targetPath)
					continue
				}
				link, err := os.Readlink(targetPath)
				if err != nil {
					return err
				}
				if link != sourcePath {
					ui.Warn("preserving unrelated generated-skill link " + targetPath)
					continue
				}
				if err := os.Remove(targetPath); err != nil {
					return err
				}
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			if err := os.Symlink(sourcePath, targetPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func unfoldConfig(repo, home string) error {
	destination := filepath.Join(home, ".config")
	info, err := os.Lstat(destination)
	if errors.Is(err, os.ErrNotExist) {
		return os.MkdirAll(destination, 0o700)
	}
	if err != nil {
		return err
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		if info.IsDir() {
			return nil
		}
		return fmt.Errorf("%s is not a directory", destination)
	}

	target, err := filepath.EvalSymlinks(destination)
	if err != nil {
		return fmt.Errorf("resolve folded ~/.config: %w", err)
	}
	targetInfo, err := os.Stat(target)
	if err != nil || !targetInfo.IsDir() {
		return fmt.Errorf("folded ~/.config target %s is not a directory", target)
	}

	managed, err := managedConfigNames(filepath.Join(repo, ".config"))
	if err != nil {
		return err
	}
	backup := filepath.Join(home, ".dotfiles-backups", time.Now().Format("20060102-150405"), "config-runtime")
	entries, err := os.ReadDir(target)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(backup, 0o700); err != nil {
		return err
	}
	// Copy (rather than move) unmanaged data first. It remains in the old
	// checkout until Stow succeeds, and the backup is an additional recovery
	// point if the host is interrupted during the migration.
	for _, entry := range entries {
		if managed[entry.Name()] {
			continue
		}
		if err := copyTree(filepath.Join(target, entry.Name()), filepath.Join(backup, entry.Name())); err != nil {
			return fmt.Errorf("back up ~/.config/%s: %w", entry.Name(), err)
		}
	}
	if err := os.Remove(destination); err != nil {
		return fmt.Errorf("remove folded ~/.config symlink: %w", err)
	}
	if err := os.Mkdir(destination, 0o700); err != nil {
		return err
	}
	for _, entry := range entries {
		if managed[entry.Name()] {
			continue
		}
		if err := copyTree(filepath.Join(backup, entry.Name()), filepath.Join(destination, entry.Name())); err != nil {
			return fmt.Errorf("restore ~/.config/%s: %w", entry.Name(), err)
		}
	}
	ui.OK("migrated folded ~/.config; preserved unmanaged runtime data in " + backup)
	return nil
}

func managedConfigNames(source string) (map[string]bool, error) {
	entries, err := os.ReadDir(source)
	if err != nil {
		return nil, err
	}
	names := make(map[string]bool, len(entries))
	for _, entry := range entries {
		names[entry.Name()] = true
	}
	return names, nil
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
		return os.Symlink(link, destination)
	}
	if !info.IsDir() {
		return copyFile(source, destination, info.Mode())
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
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func IsManagedConfigPath(repo, configPath string) bool {
	relative, err := filepath.Rel(filepath.Join(repo, ".config"), configPath)
	return err == nil && relative != "." && !strings.HasPrefix(relative, "..")
}
