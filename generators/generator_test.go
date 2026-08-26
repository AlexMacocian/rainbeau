package generators

import (
	"os"
	"path/filepath"
	"testing"
)

// A dangling symlink is invisible to os.Stat but still blocks MkdirAll with
// EEXIST. This is what a dotfiles repo leaves behind when it retires a config
// directory the user still has linked.
func TestEnsureOutputDirectoryClearsDanglingSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, ".config", "dunst")

	if err := os.MkdirAll(filepath.Join(root, ".config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "deleted-repo", "dunst"), target); err != nil {
		t.Fatal(err)
	}

	if err := ensureOutputDirectory(root, target); err != nil {
		t.Fatalf("ensureOutputDirectory: %v", err)
	}

	info, err := os.Lstat(target)
	if err != nil {
		t.Fatalf("target missing after ensure: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("dangling symlink was not replaced with a real directory")
	}
	if !info.IsDir() {
		t.Error("target is not a directory")
	}
}

// The inverse: a symlink that still resolves is the normal dotfiles pattern
// (~/.config/kitty -> repo/.config/kitty). Generated files must be written
// through it, so it must survive untouched.
func TestEnsureOutputDirectoryKeepsLiveSymlink(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo", "kitty")
	target := filepath.Join(root, ".config", "kitty")

	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(repo, target); err != nil {
		t.Fatal(err)
	}

	if err := ensureOutputDirectory(root, target); err != nil {
		t.Fatalf("ensureOutputDirectory: %v", err)
	}

	info, err := os.Lstat(target)
	if err != nil {
		t.Fatalf("target missing after ensure: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("live symlink was replaced; generated configs would stop reaching the repo")
	}

	// Writing through the link must land in the repo.
	if err := os.WriteFile(filepath.Join(target, "kitty.conf"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repo, "kitty.conf")); err != nil {
		t.Errorf("write did not reach the repo through the symlink: %v", err)
	}
}

// A plain file where a directory belongs is cleared, the original behaviour.
func TestEnsureOutputDirectoryReplacesFile(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, ".config", "wofi")

	if err := os.MkdirAll(filepath.Join(root, ".config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := ensureOutputDirectory(root, target); err != nil {
		t.Fatalf("ensureOutputDirectory: %v", err)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Error("file was not replaced with a directory")
	}
}

// The ordinary case: nothing in the way.
func TestEnsureOutputDirectoryCreatesMissing(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, ".config", "omni-shell")

	if err := ensureOutputDirectory(root, target); err != nil {
		t.Fatalf("ensureOutputDirectory: %v", err)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Error("directory was not created")
	}
}
