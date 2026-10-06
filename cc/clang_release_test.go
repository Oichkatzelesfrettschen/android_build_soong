// SPDX-License-Identifier: Apache-2.0

package cc

import (
	"strings"
	"testing"

	"android/soong/android"
)

// The archive spelling and the module-cflag pass manager selector follow the
// Clang release; the default release keeps its command lines.
func TestClangReleaseCommandLines(t *testing.T) {
	blueprint := `
		cc_library_static {
			name: "libstatic",
			srcs: ["static.cpp"],
			cflags: ["-fno-experimental-new-pass-manager"],
		}
	`
	for _, tc := range []struct {
		release      string
		arFlag       string
		wantSelector bool
	}{
		{"", " -format=gnu", true},
		{"22", " --format=gnu", false},
	} {
		env := map[string]string{}
		if tc.release != "" {
			env["LLVM_RELEASE_VERSION"] = tc.release
		}
		config := TestConfig(t.TempDir(), android.Android, env, blueprint, nil)
		context := testCcWithConfig(t, config)
		lib := context.ModuleForTests("libstatic", "android_arm_armv7-a-neon_static")

		if ar := lib.Rule("ar").Args["arFlags"]; !strings.Contains(ar, tc.arFlag) {
			t.Errorf("release %q: arFlags %q lacks %q", tc.release, ar, tc.arFlag)
		}
		cc := strings.Join(lib.Module().(*Module).flags.Local.CFlags, " ")
		if got := strings.Contains(cc, "-fno-experimental-new-pass-manager"); got != tc.wantSelector {
			t.Errorf("release %q: cFlags selector present=%t, want %t: %s", tc.release, got, tc.wantSelector, cc)
		}
	}
}
