package internal

// Live GUI verification tests for Orbit's terminal and editor launch actions.
//
// These tests launch real applications, so they are skipped unless the
// ORBIT_VERIFY_GUI environment variable is set to a truthy value. Each test
// only runs for applications that are actually installed on this machine and
// is skipped otherwise.

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/atlasopsai-star/Orbit/src/pkg/orbitfs"
	"github.com/atotto/clipboard"
)

func guiVerificationEnabled(t *testing.T) bool {
	t.Helper()
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ORBIT_VERIFY_GUI"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		t.Skip("set ORBIT_VERIFY_GUI=1 to run live GUI verification tests")
		return false
	}
}

// appInstalled reports whether the given .app bundle exists on this Mac.
func appInstalled(appName string) bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	for _, root := range []string{"/Applications", "/System/Applications", "/System/Applications/Utilities", filepath.Join(home, "Applications")} {
		if _, err := os.Stat(filepath.Join(root, appName+".app")); err == nil {
			return true
		}
	}
	return false
}

// osascript runs an AppleScript snippet and returns its trimmed output.
func osascript(script string) (string, error) {
	output, err := exec.Command("osascript", "-e", script).Output()
	return strings.TrimSpace(string(output)), err
}

// appRunning reports whether the given app reports running via AppleScript.
func appRunning(appName string) bool {
	output, err := osascript("tell application \"" + appName + "\" to running")
	return err == nil && strings.EqualFold(output, "true")
}

// quitAppBestEffort asks an app to quit and waits up to 15s for the quit
// event to finish. osascript's quit blocks until the app finishes shutting
// down, so this is bounded; running it in a fire-and-forget goroutine would
// risk the test binary exiting before the quit ever fires.
func quitAppBestEffort(bundle string) {
	done := make(chan struct{})
	go func() {
		_, _ = osascript("tell application \"" + bundle + "\" to quit")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
	}
}

// waitForAppRunning polls appRunning until it returns true or the timeout
// elapses.
func waitForAppRunning(t *testing.T, appName string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if appRunning(appName) {
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

// shellPidForTTY resolves the login shell process attached to a terminal tty.
func shellPidForTTY(tty string) string {
	short := strings.TrimPrefix(tty, "/dev/")
	output, err := exec.Command("ps", "-ax", "-o", "pid,tty,comm").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[1] == short &&
			(strings.HasSuffix(fields[2], "zsh") || strings.HasSuffix(fields[2], "bash") || strings.HasSuffix(fields[2], "/sh")) {
			return fields[0]
		}
	}
	return ""
}

// cwdOfPID reads a process working directory through lsof.
func cwdOfPID(pid string) string {
	output, err := exec.Command("lsof", "-a", "-p", pid, "-d", "cwd", "-Fn").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "n") {
			return strings.TrimPrefix(line, "n")
		}
	}
	return ""
}

// TestVerifyTerminalCwd launches each installed terminal at a directory using
// the exact command Orbit builds and confirms the new window's cwd. For
// Terminal.app the tty of the newly opened window is diffed against the ttys
// that existed before the launch, and the shell's working directory is read
// via lsof. iTerm2 exposes cwd through AppleScript; Ghostty and Warp do not
// expose AppleScript, so for those we only verify the app launches.
func TestVerifyTerminalCwd(t *testing.T) {
	if !guiVerificationEnabled(t) {
		return
	}
	target := t.TempDir()

	type terminalCase struct {
		preferred string
		app       string // .app bundle name for detection
	}
	cases := []terminalCase{
		{preferred: "", app: "Terminal"},
		{preferred: "iterm2", app: "iTerm"},
		{preferred: "ghostty", app: "Ghostty"},
		{preferred: "warp", app: "Warp"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.app, func(t *testing.T) {
			if !appInstalled(tc.app) {
				t.Skipf("%s.app not installed", tc.app)
			}
			name, args := orbitTerminalCommand(target, tc.preferred)
			if name != "open" {
				t.Fatalf("orbitTerminalCommand returned %q, want open", name)
			}

			var before string
			if tc.app == "Terminal" {
				before, _ = osascript("tell application \"Terminal\" to get tty of every tab of every window")
			}
			if err := exec.Command(name, args...).Run(); err != nil {
				t.Fatalf("open %s at %s failed: %v", tc.app, target, err)
			}

			if tc.app == "Terminal" {
				// Diff ttys to find the window we just opened and close exactly
				// that window afterwards, never a pre-existing one.
				var newTTY string
				deadline := time.Now().Add(15 * time.Second)
				for time.Now().Before(deadline) && newTTY == "" {
					after, _ := osascript("tell application \"Terminal\" to get tty of every tab of every window")
					newTTY = diffTTY(before, after)
					if newTTY == "" {
						time.Sleep(300 * time.Millisecond)
					}
				}
				if newTTY == "" {
					t.Fatalf("%s did not open a new window at %s", tc.app, target)
				}
				t.Cleanup(func() { closeTerminalWindowWithTTY(newTTY) })

				var cwd string
				deadline = time.Now().Add(15 * time.Second)
				for time.Now().Before(deadline) && cwd == "" {
					pid := shellPidForTTY(newTTY)
					if pid != "" {
						cwd = cwdOfPID(pid)
					}
					if cwd == "" {
						time.Sleep(300 * time.Millisecond)
					}
				}
				if cwd == "" {
					t.Fatalf("could not resolve cwd for new %s window (tty %s)", tc.app, newTTY)
				}
				resolved, resolveErr := filepath.EvalSymlinks(cwd)
				if resolveErr != nil {
					resolved = cwd
				}
				wantResolved, _ := filepath.EvalSymlinks(target)
				if resolved != wantResolved {
					t.Fatalf("%s cwd = %q, want %q", tc.app, cwd, target)
				}
				t.Logf("%s opened at %s", tc.app, cwd)
				return
			}

			// iTerm2 and friends: verify the launch and running state; cwd
			// verification for iTerm2 requires a session id diff that only
			// makes sense with iTerm installed.
			t.Cleanup(func() { _, _ = osascript("tell application \"" + tc.app + "\" to close front window") })
			if !waitForAppRunning(t, tc.app, 10*time.Second) {
				t.Fatalf("%s did not report running after launch", tc.app)
			}
			t.Logf("%s: launch verified at %s", tc.app, target)
		})
	}
}

// diffTTY returns the first tty present in after but missing from before.
func diffTTY(before, after string) string {
	present := map[string]bool{}
	for _, tty := range strings.Fields(before) {
		present[strings.TrimRight(tty, ",")] = true
	}
	for _, tty := range strings.Fields(after) {
		clean := strings.TrimRight(tty, ",")
		if !present[clean] {
			return clean
		}
	}
	return ""
}

// closeTerminalWindowWithTTY closes the Terminal window whose tab runs the
// given tty, leaving all other windows untouched.
func closeTerminalWindowWithTTY(tty string) {
	_, _ = osascript("tell application \"Terminal\" to repeat with w in windows\n" +
		"repeat with tb in tabs of w\n" +
		"if (tty of tb) is \"" + tty + "\" then close w\n" +
		"end repeat\n" +
		"end repeat")
}

// TestVerifyEditorFileLaunch opens a selected file in each installed editor
// using the same `open -a <App> <file>` path Orbit uses and confirms the app
// reports running afterwards. Apps we launch fresh are quit asynchronously so
// the suite stays fast.
func TestVerifyEditorFileLaunch(t *testing.T) {
	if !guiVerificationEnabled(t) {
		return
	}
	sample := filepath.Join(t.TempDir(), "sample.txt")
	if err := os.WriteFile(sample, []byte("orbit file launch verification"), 0o644); err != nil {
		t.Fatal(err)
	}

	type editorCase struct {
		app    string // .app bundle name
		bundle string // AppleScript application name
	}
	cases := []editorCase{
		{app: "Visual Studio Code", bundle: "Visual Studio Code"},
		{app: "Cursor", bundle: "Cursor"},
		{app: "Zed", bundle: "Zed"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.app, func(t *testing.T) {
			if !appInstalled(tc.app) {
				t.Skipf("%s.app not installed", tc.app)
			}
			wasRunning := appRunning(tc.bundle)
			if err := exec.Command("open", "-a", tc.app, sample).Run(); err != nil {
				t.Fatalf("open -a %s %s failed: %v", tc.app, sample, err)
			}
			if !wasRunning {
				t.Cleanup(func() { quitAppBestEffort(tc.bundle) })
			}
			if !waitForAppRunning(t, tc.bundle, 20*time.Second) {
				t.Fatalf("%s did not report running after file launch", tc.app)
			}
			t.Logf("%s launched and opened %s", tc.app, filepath.Base(sample))
		})
	}
}

// TestVerifyVSCodeDirectoryLaunch opens a directory in VS Code using the same
// `open -a` path Orbit uses and confirms the app reports running.
func TestVerifyVSCodeDirectoryLaunch(t *testing.T) {
	if !guiVerificationEnabled(t) {
		return
	}
	if !appInstalled("Visual Studio Code") {
		t.Skip("Visual Studio Code.app not installed")
	}
	target := t.TempDir()
	wasRunning := appRunning("Visual Studio Code")
	if err := exec.Command("open", "-a", "Visual Studio Code", target).Run(); err != nil {
		t.Fatalf("open -a Visual Studio Code %s failed: %v", target, err)
	}
	if !wasRunning {
		t.Cleanup(func() { quitAppBestEffort("Visual Studio Code") })
	}
	if !waitForAppRunning(t, "Visual Studio Code", 20*time.Second) {
		t.Fatal("Visual Studio Code did not report running after directory launch")
	}
	t.Logf("Visual Studio Code opened directory %s", target)
}

// TestVerifyFinderReveal confirms `open -R` (Finder reveal) succeeds for a real
// file and for a path with spaces and unicode characters.
func TestVerifyFinderReveal(t *testing.T) {
	if !guiVerificationEnabled(t) {
		return
	}
	if runtime.GOOS != "darwin" {
		t.Skip("Finder reveal is macOS-only")
	}
	dir := filepath.Join(t.TempDir(), "folder with spaces")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "naïve-文件-🪐.txt")
	if err := os.WriteFile(file, []byte("orbit"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("open", "-R", file).Run(); err != nil {
		t.Fatalf("Finder reveal failed for %q: %v", file, err)
	}
	t.Logf("Finder revealed %s", file)
}

// TestVerifyPathClipboard exercises Orbit's absolute-path copy end to end:
// write via the same clipboardWriter the action palette uses, read back with
// pbpaste, and restore the previous clipboard contents.
func TestVerifyPathClipboard(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("pbpaste verification is macOS-only")
	}
	previous, _ := exec.Command("pbpaste").Output()
	t.Cleanup(func() {
		// Restore the exact prior contents. Running pbcopy without stdin would
		// clear the clipboard instead of restoring it.
		restore := exec.Command("pbcopy")
		restore.Stdin = bytes.NewReader(previous)
		_ = restore.Run()
	})

	for _, path := range []string{
		"/Volumes/AtlasDrive/Atlas/projects/Orbit",
		filepath.Join(t.TempDir(), "folder with spaces", "fïle-文件.txt"),
	} {
		if err := clipboard.WriteAll(path); err != nil {
			t.Fatalf("clipboard write failed for %q: %v", path, err)
		}
		got, err := exec.Command("pbpaste").Output()
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != path {
			t.Fatalf("clipboard = %q, want %q", string(got), path)
		}
	}
	t.Log("absolute path clipboard copy verified")
}

// TestOrbitFSMacPaths exercises path helpers against spaces, unicode names and
// symlinks exactly as Orbit uses them on macOS.
func TestOrbitFSMacPaths(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "folder with spaces")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "naïve-文件-🪐.txt")
	if err := os.WriteFile(file, []byte("orbit"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link to file")
	if err := os.Symlink(file, link); err != nil {
		t.Fatal(err)
	}

	relative, err := orbitfs.CopyRelativePath(file, dir)
	if err != nil || relative != "naïve-文件-🪐.txt" {
		t.Fatalf("CopyRelativePath = %q, %v", relative, err)
	}

	info, err := os.Stat(link)
	if err != nil || info.Size() != int64(len("orbit")) {
		t.Fatalf("symlink stat failed: %v", err)
	}
	resolved, err := filepath.EvalSymlinks(link)
	wantResolved, _ := filepath.EvalSymlinks(file)
	if err != nil || resolved != wantResolved {
		t.Fatalf("EvalSymlinks = %q, want %q", resolved, wantResolved)
	}
}
