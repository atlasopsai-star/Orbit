package actions

import "testing"

func TestDefaultAvailabilityAndFiltering(t *testing.T) {
	items := Default(Context{SelectedPath: "/tmp/demo.go", CurrentDirectory: "/tmp", IsMac: true, AvailableApps: map[string]bool{"Cursor": true}})
	filtered := Filter(items, "cursor")
	if len(filtered) != 1 || filtered[0].ID != "open-cursor" {
		t.Fatalf("cursor filter = %#v", filtered)
	}
	for _, item := range items {
		if item.ID == "folder-size" {
			t.Fatal("folder-size should not apply to a file")
		}
	}
}

func TestDefaultHidesUnavailableEditorsAndExtract(t *testing.T) {
	items := Default(Context{SelectedPath: "/tmp/demo.go", CurrentDirectory: "/tmp", IsMac: true})
	for _, item := range items {
		if item.ID == "open-cursor" || item.ID == "open-vscode" || item.ID == "open-zed" || item.ID == "open-xcode" || item.ID == "extract" {
			t.Fatalf("unexpected unavailable action %q", item.ID)
		}
	}
}
