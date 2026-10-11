package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func defaultsFixture() Config {
	return Config{
		SchemaVersion: CurrentSchemaVersion,
		DownloadsDir:  filepath.Join("/home", "user", "Downloads"),
		YtDlpChannel:  DefaultYtDlpChannel,
		ProxyStrategy: DefaultProxyStrategy,
		DepSources:    map[string]string{"yt-dlp": "auto"},
	}
}

func TestLoadMissingFileUsesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")

	store, err := Load(path, defaultsFixture())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got, want := store.Get(), defaultsFixture(); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestUpdatePersistsChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")

	store, err := Load(path, defaultsFixture())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if err := store.Update(func(c *Config) { c.YtDlpChannel = "nightly" }); err != nil {
		t.Fatalf("Update: %v", err)
	}

	reloaded, err := Load(path, defaultsFixture())
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := reloaded.Get().YtDlpChannel; got != "nightly" {
		t.Fatalf("channel = %q, want nightly", got)
	}
}

func TestUpdateRejectsInvalidChannelAndKeepsState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")

	store, err := Load(path, defaultsFixture())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if err := store.Update(func(c *Config) { c.YtDlpChannel = "bogus" }); err == nil {
		t.Fatal("Update: ожидалась ошибка для неизвестного канала")
	}
	if got := store.Get().YtDlpChannel; got != DefaultYtDlpChannel {
		t.Fatalf("channel = %q, want %q", got, DefaultYtDlpChannel)
	}
}

func TestLoadPartialFileFillsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"downloadsDir":"/custom"}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	store, err := Load(path, defaultsFixture())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := store.Get()
	if got.DownloadsDir != "/custom" {
		t.Fatalf("downloadsDir = %q, want /custom", got.DownloadsDir)
	}
	if got.YtDlpChannel != DefaultYtDlpChannel {
		t.Fatalf("channel = %q, want %q", got.YtDlpChannel, DefaultYtDlpChannel)
	}
	if got.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf("schemaVersion = %d, want %d", got.SchemaVersion, CurrentSchemaVersion)
	}
}
