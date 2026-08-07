package setup

import (
	"reflect"
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
