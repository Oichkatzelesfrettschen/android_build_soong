package config

import (
	"reflect"
	"testing"
)

func TestClangNewWarningCflags(t *testing.T) {
	r563880 := clangNewWarningBlocks[0].cflags
	r574158 := clangNewWarningBlocks[1].cflags
	r584948 := clangNewWarningBlocks[2].cflags
	cat := func(lists ...[]string) []string {
		var out []string
		for _, l := range lists {
			out = append(out, l...)
		}
		return out
	}
	for _, tc := range []struct {
		version string
		want    []string
	}{
		{"", nil},
		{"clang-r536225", nil},
		{"clang-r563880", r563880},
		{"clang-r563880c", r563880},
		{"clang-r574158", cat(r563880, r574158)},
		{"clang-r584948", cat(r563880, r574158, r584948)},
		{"clang-r584948b", cat(r563880, r574158, r584948)},
		{"clang-r596125", cat(r563880, r574158, r584948)},
		{"clang-dev", nil},
	} {
		if got := clangNewWarningCflags(tc.version); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("clangNewWarningCflags(%q) = %q, want %q", tc.version, got, tc.want)
		}
	}
}

func TestClangNewWarningDirCflags(t *testing.T) {
	virt := []string{"-Wno-error=unnecessary-virtual-specifier"}
	for _, tc := range []struct {
		version, dir string
		want         []string
	}{
		{"clang-r584948", "external/angle", virt},
		{"clang-r584948", "external/angle/src/libANGLE", virt},
		{"clang-r584948b", "external/webrtc", virt},
		{"clang-r584948", "external/angleproject", nil},
		{"clang-r584948", "external/icu", nil},
		{"clang-r536225", "external/angle", nil},
		{"", "external/angle", nil},
	} {
		if got := ClangNewWarningDirCflags(tc.version, tc.dir); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("ClangNewWarningDirCflags(%q, %q) = %q, want %q", tc.version, tc.dir, got, tc.want)
		}
	}
}

func TestClangWarningInventoryCflags(t *testing.T) {
	got := clangWarningInventoryCflags([]string{
		"-Wno-character-conversion",
		"-Wno-error=uninitialized-const-pointer",
	})
	want := []string{
		"-Wno-error=character-conversion",
		"-Wno-error=uninitialized-const-pointer",
		"-Wno-error",
		"-ferror-limit=0",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("clangWarningInventoryCflags = %q, want %q", got, want)
	}
	if got := clangWarningInventoryCflags(nil); !reflect.DeepEqual(got, []string{"-Wno-error", "-ferror-limit=0"}) {
		t.Errorf("clangWarningInventoryCflags(nil) = %q", got)
	}
}
