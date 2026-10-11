package proxy

import (
	"path/filepath"
	"testing"
)

func TestStoreAddPersistsAndDeduplicates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "proxies.json")
	store, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(store.List()) != 0 {
		t.Fatal("пул должен быть пустым")
	}

	list := ParseList("socks5://1.2.3.4:1080\nsocks5://5.6.7.8:1081").Proxies
	added, err := store.Add(list)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if added != 2 {
		t.Fatalf("added = %d, want 2", added)
	}

	// Повторный импорт тех же прокси ничего не добавляет.
	added, err = store.Add(list)
	if err != nil {
		t.Fatalf("повторный Add: %v", err)
	}
	if added != 0 {
		t.Fatalf("повторный added = %d, want 0", added)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := len(reloaded.List()); got != 2 {
		t.Fatalf("после перезагрузки %d, want 2", got)
	}
}

func TestStoreToggleRemoveClear(t *testing.T) {
	store, err := Load(filepath.Join(t.TempDir(), "proxies.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	list := ParseList("socks5://1.2.3.4:1080").Proxies
	if _, err := store.Add(list); err != nil {
		t.Fatalf("Add: %v", err)
	}
	id := list[0].ID

	if err := store.SetEnabled(id, false); err != nil {
		t.Fatalf("SetEnabled: %v", err)
	}
	if store.List()[0].Enabled {
		t.Error("прокси должен быть выключен")
	}

	if err := store.Remove(id); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if len(store.List()) != 0 {
		t.Error("прокси не удалён")
	}
	if err := store.Remove(id); err == nil {
		t.Error("повторное удаление — ожидалась ошибка")
	}
	if err := store.SetEnabled(id, true); err == nil {
		t.Error("включение отсутствующего — ожидалась ошибка")
	}

	if _, err := store.Add(ParseList("socks5://9.9.9.9:1080").Proxies); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := store.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if len(store.List()) != 0 {
		t.Error("пул не очищен")
	}
}

func TestStoreListReturnsCopy(t *testing.T) {
	store, err := Load(filepath.Join(t.TempDir(), "proxies.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := store.Add(ParseList("socks5://1.2.3.4:1080").Proxies); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got := store.List()
	got[0].Host = "изменён"
	if store.List()[0].Host != "1.2.3.4" {
		t.Error("List должен возвращать копию")
	}
}
