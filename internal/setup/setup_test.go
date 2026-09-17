package setup

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDefaultsAddT3ServiceOnlyWhenRequested(t *testing.T) {
	o := Options{Profile: "server", Defaults: true, T3Channel: "nightly"}
	if err := o.Normalize(); err != nil {
		t.Fatal(err)
	}
	if err := o.Choose(); err != nil {
		t.Fatal(err)
	}
	if !contains(o.Components, "core") || !contains(o.Components, "t3-service") {
		t.Fatalf("components = %#v, want default setup plus t3-service", o.Components)
	}
}

func TestDefaultComponentsSkipDesktopOnServer(t *testing.T) {
	o := Options{Profile: "server"}
	if contains(o.DefaultComponents(), "desktop") {
		t.Fatalf("server defaults should not include desktop: %#v", o.DefaultComponents())
	}
}

func TestPackageInstallArgsUseOnlyAvailablePackages(t *testing.T) {
	available := []string{"git", "fd"}
	got := packageInstallArgs("brew", available)
	want := []string{"install", "git", "fd"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("packageInstallArgs() = %#v, want %#v", got, want)
	}
}

func TestReplacePackageUsesPlatformSpecificName(t *testing.T) {
	got := replacePackage([]string{"git", "fd", "ripgrep"}, "fd", "fd-find")
	want := []string{"git", "fd-find", "ripgrep"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("replacePackage() = %#v, want %#v", got, want)
	}
}

func TestFNMEnvironmentUsesSupportedShell(t *testing.T) {
	environment := fnmEnvironment()
	if (!strings.Contains(environment, "--shell zsh") && !strings.Contains(environment, "--shell bash")) || strings.Contains(environment, "--shell sh") {
		t.Fatalf("fnmEnvironment() = %q, want Zsh", environment)
	}
}

func TestQuietCommandPreservesFailuresWithoutSuccessOutput(t *testing.T) {
	command := quietCommand("npx -y skills add owner/repo -y")
	for _, want := range []string{"mktemp", ">\"$log\" 2>&1", "cat \"$log\" >&2", "exit 1"} {
		if !strings.Contains(command, want) {
			t.Fatalf("quietCommand() = %q, want %q", command, want)
		}
	}
}

func TestNeedsZshLoginShellRecognizesSystemAndHomebrewZsh(t *testing.T) {
	if needsZshLoginShell("/bin/zsh", "/opt/homebrew/bin/zsh") {
		t.Fatal("two Zsh paths should not trigger a login-shell change")
	}
	if !needsZshLoginShell("/bin/bash", "/usr/bin/zsh") {
		t.Fatal("Bash should trigger a login-shell change")
	}
}

func TestShellNonInteractiveInstallsPluginsIdempotently(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh unavailable")
	}
	home := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, ".oh-my-zsh"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SHELL", "/bin/bash")
	t.Setenv("ZSH_CUSTOM", "")
	calls := 0
	runner := Runner{Run: func(name string, args ...string) error {
		if name != "git" || args[0] != "clone" {
			t.Fatalf("unexpected command: %s %v", name, args)
		}
		calls++
		return os.MkdirAll(args[len(args)-1], 0o700)
	}}
	for i := 0; i < 2; i++ {
		if err := runner.installShell(Options{Home: home, NonInteractive: true}); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 2 {
		t.Fatalf("got %d plugin installs, want 2", calls)
	}
}

func TestInstallerFailuresStopFollowingCommands(t *testing.T) {
	for _, script := range []string{"#!/bin/sh\nexit 22\n", "#!/bin/sh\nwhile [ $# -gt 0 ]; do if [ \"$1\" = -o ]; then shift; printf 'exit 19\\n' > \"$1\"; exit 0; fi; shift; done\n"} {
		home := t.TempDir()
		bin := t.TempDir()
		curl := writeFixture(t, bin, "curl", script)
		if err := os.Chmod(curl, 0o700); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
		t.Setenv("HOME", home)
		runner := Runner{Run: func(name string, args ...string) error { return exec.Command(name, args...).Run() }}
		marker := filepath.Join(home, "unexpected")
		command := installerCommand("https://example.invalid/install", "sh") + "; touch " + shellPath(marker)
		if err := runner.shell(command); err == nil {
			t.Fatal("installer failure swallowed")
		}
		if _, err := os.Stat(marker); !os.IsNotExist(err) {
			t.Fatal("later command ran")
		}
	}
}

func TestFailedFNMStopsNodeWork(t *testing.T) {
	home := t.TempDir()
	bin := t.TempDir()
	fnm := writeFixture(t, bin, "fnm", "#!/bin/sh\nexit 7\n")
	if err := os.Chmod(fnm, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	runner := Runner{Run: func(name string, args ...string) error { return exec.Command(name, args...).Run() }}
	if err := runner.nodeShell("touch " + shellPath(filepath.Join(home, "unexpected"))); err == nil {
		t.Fatal("failed fnm ignored")
	}
	if _, err := os.Stat(filepath.Join(home, "unexpected")); !os.IsNotExist(err) {
		t.Fatal("Node work ran after failed fnm")
	}
}

func TestInitializeOnlyMissingSubmodules(t *testing.T) {
	repo := t.TempDir()
	writeFixture(t, repo, ".gitmodules", "fixture")
	bin := t.TempDir()
	git := writeFixture(t, bin, "git", "#!/bin/sh\nprintf '%s\\n' '-123456 missing' '+123456 dirty' ' 123456 initialized'\n")
	if err := os.Chmod(git, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	var calls [][]string
	runner := Runner{Run: func(name string, args ...string) error {
		calls = append(calls, append([]string{name}, args...))
		return nil
	}}
	if err := runner.initializeSubmodules(repo); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0][len(calls[0])-1] != "missing" {
		t.Fatalf("unexpected submodule updates: %v", calls)
	}
}

func TestNormalizeMakesPathsAbsolute(t *testing.T) {
	o := Options{Repo: ".", Home: "."}
	if err := o.Normalize(); err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(o.Repo) || !filepath.IsAbs(o.Home) {
		t.Fatalf("paths not absolute: %+v", o)
	}
}
