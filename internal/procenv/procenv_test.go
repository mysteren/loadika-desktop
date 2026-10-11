package procenv

import (
	"reflect"
	"strings"
	"testing"
)

func TestEnvFromWithoutAppImage(t *testing.T) {
	in := []string{"PATH=/usr/bin", "PYTHONHOME=/custom", "HOME=/home/u"}
	got := envFrom(in, "")
	if !reflect.DeepEqual(got, in) {
		t.Fatalf("вне AppImage окружение не должно меняться; got %v", got)
	}
}

func TestWithPythonEnvSetsAndReplaces(t *testing.T) {
	in := []string{"PATH=/usr/bin", "PYTHONIOENCODING=latin-1", "HOME=/home/u"}
	got := withPythonEnv(in)

	want := map[string]string{"PYTHONUTF8": "1", "PYTHONIOENCODING": "utf-8", "PYTHONUNBUFFERED": "1"}
	gotMap := map[string]string{}
	for _, kv := range got {
		key, val, _ := strings.Cut(kv, "=")
		gotMap[key] = val
	}
	for key, w := range want {
		if gotMap[key] != w {
			t.Errorf("%s = %q, want %q", key, gotMap[key], w)
		}
	}
	// Прочие переменные должны сохраняться.
	if gotMap["PATH"] != "/usr/bin" || gotMap["HOME"] != "/home/u" {
		t.Errorf("withPythonEnv потерял переменные: %v", got)
	}

	// Исходный срез не должен изменяться.
	if in[1] != "PYTHONIOENCODING=latin-1" {
		t.Errorf("withPythonEnv изменил вход: %v", in)
	}
}

func TestEnvFromAppImage(t *testing.T) {
	in := []string{
		"PATH=/tmp/.mount_x/usr/bin:/usr/bin",
		"PYTHONHOME=/tmp/.mount_x/usr/",
		"PYTHONPATH=/tmp/.mount_x/usr/share/pyshared/",
		"PYTHONSTARTUP=/tmp/.mount_x/usr/startup.py",
		"LD_LIBRARY_PATH=/tmp/.mount_x/usr/lib",
		"HOME=/home/u",
	}
	want := []string{
		"PATH=/tmp/.mount_x/usr/bin:/usr/bin",
		"HOME=/home/u",
	}
	got := envFrom(in, "/tmp/.mount_x")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("в AppImage «вредные» переменные должны удаляться; got %v, want %v", got, want)
	}
}
