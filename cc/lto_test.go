package cc

import (
	"strings"
	"testing"

	"android/soong/android"
)

func TestGlobalThinLtoOptIn(t *testing.T) {
	blueprint := `
		cc_library_shared { name: "libglobal", srcs: ["global.c"] }
		cc_library_shared { name: "libnever", srcs: ["never.c"], lto: { never: true } }
		cc_library_shared { name: "libfull", srcs: ["full.c"], lto: { full: true } }
		cc_library_static { name: "libstatic", srcs: ["static.c"] }
		cc_test { name: "global_test", srcs: ["test.c"], gtest: false }
	`
	for _, enabled := range []bool{false, true} {
		environment := map[string]string{}
		if enabled {
			environment["GLOBAL_THINLTO"] = "true"
		}
		config := TestConfig(buildDir, android.Android, environment, blueprint, nil)
		context := testCcWithConfig(t, config)
		for _, module := range []struct {
			name    string
			variant string
			thin    bool
		}{
			{"libglobal", "android_arm_armv7-a-neon_shared", enabled},
			{"libnever", "android_arm_armv7-a-neon_shared", false},
			{"libfull", "android_arm_armv7-a-neon_shared", false},
			{"libstatic", "android_arm_armv7-a-neon_static", false},
			{"global_test", "android_arm_armv7-a-neon", false},
		} {
			compiled := context.ModuleForTests(module.name, module.variant).Module().(*Module)
			compileFlags := strings.Join(compiled.flags.Local.CFlags, " ")
			containsThin := strings.Contains(compileFlags, "-flto=thin")
			if containsThin != module.thin {
				t.Errorf("GLOBAL_THINLTO=%t %s compile flags %q: ThinLTO=%t, want %t", enabled, module.name, compileFlags, containsThin, module.thin)
			}
		}
	}
}
