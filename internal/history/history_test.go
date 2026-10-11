package history

import (
	"path/filepath"
	"testing"
)

func TestLoadMissingFileIsEmpty(t *testing.T) {
	store, err := Load(filepath.Join(t.TempDir(), "history.json"), DefaultMaxEntries)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := store.List(); len(got) != 0 {
		t.Fatalf("история не пуста: %+v", got)
	}
}

func TestAddPersistsNewestFirst(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")

	store, err := Load(path, DefaultMaxEntries)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := store.Add(Entry{ID: "1", URL: "a", State: "completed"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := store.Add(Entry{ID: "2", URL: "b", State: "failed"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got := store.List()
	if len(got) != 2 || got[0].ID != "2" || got[1].ID != "1" {
		t.Fatalf("порядок неверный: %+v", got)
	}

	reloaded, err := Load(path, DefaultMaxEntries)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	again := reloaded.List()
	if len(again) != 2 || again[0].URL != "b" {
		t.Fatalf("после перезагрузки: %+v", again)
	}
}

func TestAddTrimsToMax(t *testing.T) {
	store, err := Load(filepath.Join(t.TempDir(), "history.json"), 2)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, id := range []string{"1", "2", "3"} {
		if err := store.Add(Entry{ID: id}); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}

	got := store.List()
	if len(got) != 2 || got[0].ID != "3" || got[1].ID != "2" {
		t.Fatalf("обрезание не сработало: %+v", got)
	}
}

func TestClear(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")

	store, err := Load(path, DefaultMaxEntries)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := store.Add(Entry{ID: "1"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := store.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if got := store.List(); len(got) != 0 {
		t.Fatalf("после Clear: %+v", got)
	}

	reloaded, err := Load(path, DefaultMaxEntries)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := reloaded.List(); len(got) != 0 {
		t.Fatalf("после перезагрузки: %+v", got)
	}
}

func TestRemoveAndGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")

	store, err := Load(path, DefaultMaxEntries)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := store.Add(Entry{ID: "1", URL: "a"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := store.Add(Entry{ID: "2", URL: "b"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if e, ok := store.Get("1"); !ok || e.URL != "a" {
		t.Fatalf("Get(1) = %+v, ok=%v", e, ok)
	}
	if _, ok := store.Get("нет"); ok {
		t.Error("Get несуществующей записи должен вернуть false")
	}

	if err := store.Remove("1"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, ok := store.Get("1"); ok {
		t.Error("запись не удалена")
	}

	// Удаление сохраняется на диск.
	reloaded, err := Load(path, DefaultMaxEntries)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := reloaded.List(); len(got) != 1 || got[0].ID != "2" {
		t.Fatalf("после перезагрузки: %+v", got)
	}
}

func TestRemoveOnlyFirstMatch(t *testing.T) {
	store, err := Load(filepath.Join(t.TempDir(), "history.json"), DefaultMaxEntries)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Искусственное совпадение id: Remove должен убрать только одну запись.
	if err := store.Add(Entry{ID: "same", URL: "a"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := store.Add(Entry{ID: "same", URL: "b"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := store.Remove("same"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if got := store.List(); len(got) != 1 {
		t.Fatalf("осталось %d записей, want 1: %+v", len(got), got)
	}
}

func TestListReturnsCopy(t *testing.T) {
	store, err := Load(filepath.Join(t.TempDir(), "history.json"), DefaultMaxEntries)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := store.Add(Entry{ID: "1"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got := store.List()
	got[0].ID = "изменён"

	if store.List()[0].ID != "1" {
		t.Error("List должен возвращать копию")
	}
}
