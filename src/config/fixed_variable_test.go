package variable

import (
	"strings"
	"testing"
)

func TestOrbitFoundationPaths(t *testing.T) {
	for name, path := range map[string]string{
		"config": variablePath(SuperFileMainDir),
		"cache":  variablePath(SuperFileCacheDir),
		"data":   variablePath(SuperFileDataDir),
		"state":  variablePath(SuperFileStateDir),
	} {
		if !strings.HasSuffix(path, "/orbit") {
			t.Errorf("%s path = %q, want an Orbit namespace", name, path)
		}
	}
	if !strings.HasSuffix(LogFile, "/orbit/orbit.log") {
		t.Errorf("log path = %q, want an Orbit log", LogFile)
	}
	if EmbedThemeOrbitDarkFile != "src/orbit_config/theme/orbit-dark.toml" {
		t.Errorf("Orbit Dark theme path = %q", EmbedThemeOrbitDarkFile)
	}
	if EmbedConfigDir != "src/orbit_config" {
		t.Errorf("embedded config dir = %q", EmbedConfigDir)
	}
	if LatestVersionGithub != "https://github.com/atlasopsai-star/Orbit/releases/latest" {
		t.Errorf("latest release link = %q", LatestVersionGithub)
	}
	if OrbitReleasesAvailable {
		t.Fatal("Orbit release checks must remain disabled until the first release")
	}
}

func variablePath(path string) string {
	return path
}
