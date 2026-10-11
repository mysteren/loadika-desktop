package procenv

import (
	"reflect"
	"testing"
)

func TestEnvFromWithoutAppImage(t *testing.T) {
	in := []string{"PATH=/usr/bin", "PYTHONHOME=/custom", "HOME=/home/u"}
	got := envFrom(in, "")
	if !reflect.DeepEqual(got, in) {
		t.Fatalf("вне AppImage окружение не должно меняться; got %v", got)
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
