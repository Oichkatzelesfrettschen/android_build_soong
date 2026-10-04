package cc

import (
	"strings"
	"testing"

	"android/soong/android"
)

func prepareForGlobalThinLtoTest(enabled bool) android.FixturePreparer {
	env := map[string]string{}
	if enabled {
		env["GLOBAL_THINLTO"] = "true"
	}
	return android.GroupFixturePreparers(prepareForCcTest, android.FixtureMergeEnv(env))
}

// ltoFlagsOf joins the compile flags of one 64-bit device variant of a module.
func ltoFlagsOf(result *android.TestResult, name, variant string) string {
	return result.ModuleForTests(name, variant).Module().VariablesForTests()["cflags"]
}

// ltoVariantsOf lists the 64-bit device variants of a module.
func ltoVariantsOf(result *android.TestResult, name string) []string {
	var variants []string
	for _, variant := range result.ModuleVariantsForTests(name) {
		if strings.HasPrefix(variant, "android_arm64_armv8-a") {
			variants = append(variants, variant)
		}
	}
	return variants
}

func TestGlobalThinLtoOptIn(t *testing.T) {
	bp := `
		cc_library_shared { name: "libglobal", srcs: ["global.c"], static_libs: ["libdep"] }
		cc_library_shared { name: "libnever", srcs: ["never.c"], lto: { never: true }, static_libs: ["libneverdep"] }
		cc_library_shared { name: "libfull", srcs: ["full.c"], lto: { full: true } }
		cc_library_shared { name: "libexplicit", srcs: ["explicit.c"], lto: { thin: true } }
		cc_library_static { name: "libdep", srcs: ["dep.c"] }
		cc_library_static { name: "libneverdep", srcs: ["neverdep.c"] }
		cc_library_static { name: "libstatic", srcs: ["static.c"] }
		cc_library_shared { name: "libcfi", srcs: ["cfi.c"], sanitize: { cfi: true } }
		cc_binary { name: "static_binary", srcs: ["static_binary.c"], static_executable: true }
		cc_binary { name: "dynamic_binary", srcs: ["dynamic_binary.c"] }
		cc_binary_host { name: "host_binary", srcs: ["host.c"] }
		cc_test { name: "global_test", srcs: ["test.c"], gtest: false }
		cc_benchmark { name: "global_benchmark", srcs: ["benchmark.c"] }
	`
	for _, enabled := range []bool{false, true} {
		result := android.GroupFixturePreparers(
			prepareForGlobalThinLtoTest(enabled),
		).RunTestWithBp(t, bp)

		for _, module := range []struct {
			name    string
			variant string
			thin    bool
		}{
			{"libglobal", "android_arm64_armv8-a_shared", enabled},
			{"libexplicit", "android_arm64_armv8-a_shared", true},
			{"libnever", "android_arm64_armv8-a_shared", false},
			{"libfull", "android_arm64_armv8-a_shared", false},
			{"libcfi", "android_arm64_armv8-a_shared_cfi", false},
			{"libstatic", "android_arm64_armv8-a_static", false},
			{"static_binary", "android_arm64_armv8-a", false},
			{"dynamic_binary", "android_arm64_armv8-a", enabled},
			{"global_test", "android_arm64_armv8-a", false},
			{"global_benchmark", "android_arm64_armv8-a", false},
		} {
			flags := ltoFlagsOf(result, module.name, module.variant)
			if got := strings.Contains(flags, "-flto=thin"); got != module.thin {
				t.Errorf("GLOBAL_THINLTO=%t %s cflags %q: ThinLTO=%t, want %t", enabled, module.name, flags, got, module.thin)
			}
		}

		// A selected link unit builds its static dependencies as ThinLTO bitcode in
		// a separate variant; a module that opts out gets no such variant.
		hasThinVariant := func(name string) bool {
			for _, variant := range ltoVariantsOf(result, name) {
				if strings.HasSuffix(variant, "_lto-thin") {
					return true
				}
			}
			return false
		}
		if got := hasThinVariant("libdep"); got != enabled {
			t.Errorf("GLOBAL_THINLTO=%t libdep lto-thin variant=%t, want %t", enabled, got, enabled)
		}
		if hasThinVariant("libneverdep") {
			t.Errorf("GLOBAL_THINLTO=%t libneverdep has a lto-thin variant under a lto never module", enabled)
		}
	}
}

// With GLOBAL_THINLTO unset no module selects LTO, and the link units keep
// the flags they carry without the switch.
func TestGlobalThinLtoUnsetChangesNothing(t *testing.T) {
	bp := `
		cc_library_shared { name: "libshared", srcs: ["shared.c"], static_libs: ["libstatic"] }
		cc_library_static { name: "libstatic", srcs: ["static.c"] }
		cc_binary { name: "binary", srcs: ["binary.c"] }
		cc_benchmark { name: "benchmark", srcs: ["benchmark.c"] }
		cc_test { name: "unit_test", srcs: ["test.c"], gtest: false }
		cc_fuzz { name: "fuzzer", srcs: ["fuzz.c"], shared_libs: ["libshared"] }
	`
	result := prepareForGlobalThinLtoTest(false).RunTestWithBp(t, bp)
	for _, name := range []string{"libshared", "libstatic", "binary", "benchmark", "unit_test", "fuzzer"} {
		for _, variant := range ltoVariantsOf(result, name) {
			if strings.Contains(variant, "lto-") {
				t.Errorf("%s variant %q exists with GLOBAL_THINLTO unset", name, variant)
			}
			module := result.ModuleForTests(name, variant).Module().(*Module)
			if module.lto.Properties.GlobalThin || module.lto.ThinLTO() {
				t.Errorf("%s %s selected ThinLTO with GLOBAL_THINLTO unset", name, variant)
			}
			if flags := ltoFlagsOf(result, name, variant); strings.Contains(flags, "-flto") {
				t.Errorf("%s %s cflags %q carry -flto with GLOBAL_THINLTO unset", name, variant, flags)
			}
		}
	}
}

// The cc_binary and cc_library_shared 64-bit link rules carry the upstream
// cross-unit import limit for every LTO module, global or module-selected.
func TestGlobalThinLtoKeepsImportLimit(t *testing.T) {
	bp := `
		cc_library_shared { name: "libglobal", srcs: ["global.c"] }
		cc_library_shared { name: "libexplicit", srcs: ["explicit.c"], lto: { thin: true } }
	`
	result := prepareForGlobalThinLtoTest(true).RunTestWithBp(t, bp)
	for _, name := range []string{"libglobal", "libexplicit"} {
		ld := result.ModuleForTests(name, "android_arm64_armv8-a_shared").Rule("ld").Args["ldFlags"]
		if !strings.Contains(ld, "-flto=thin") || !strings.Contains(ld, "-import-instr-limit=5") {
			t.Errorf("%s ldflags %q lack the ThinLTO link policy", name, ld)
		}
		if strings.Contains(ld, "-inline-threshold=0") {
			t.Errorf("%s ldflags %q carry a zero inline threshold", name, ld)
		}
	}
}

// A shared library reached through a cc_fuzz target receives
// Sanitize.Fuzzer after begin() has run, so its fuzzer variant must give up
// the ThinLTO selection before ltoDepsMutator builds static dependencies for it.
func TestGlobalThinLtoSkipsFuzzerSharedLibs(t *testing.T) {
	bp := `
		cc_fuzz { name: "fuzzer", srcs: ["fuzz.c"], shared_libs: ["libfuzzed"] }
		cc_library_shared { name: "libfuzzed", srcs: ["fuzzed.c"], static_libs: ["libfuzzeddep"] }
		cc_library_static { name: "libfuzzeddep", srcs: ["fuzzeddep.c"] }
		cc_binary { name: "plain", srcs: ["plain.c"], shared_libs: ["libplain"] }
		cc_library_shared { name: "libplain", srcs: ["plain_lib.c"], static_libs: ["libplaindep"] }
		cc_library_static { name: "libplaindep", srcs: ["plaindep.c"] }
	`
	result := prepareForGlobalThinLtoTest(true).RunTestWithBp(t, bp)

	// The non-fuzzer variants of libfuzzed and libfuzzeddep keep the global
	// selection; only the variants the fuzz target links are checked.
	for _, name := range []string{"fuzzer", "libfuzzed", "libfuzzeddep"} {
		checked := 0
		for _, variant := range ltoVariantsOf(result, name) {
			if name != "fuzzer" && !strings.Contains(variant, "_fuzzer") {
				continue
			}
			checked++
			if strings.Contains(variant, "lto-") {
				t.Errorf("%s variant %q: ThinLTO variant built for a fuzz target", name, variant)
			}
			module := result.ModuleForTests(name, variant).Module().(*Module)
			if module.lto.ThinLTO() || module.lto.Properties.GlobalThin {
				t.Errorf("%s %s keeps ThinLTO under a fuzz target", name, variant)
			}
			if flags := ltoFlagsOf(result, name, variant); strings.Contains(flags, "-flto") {
				t.Errorf("%s %s cflags %q carry -flto under a fuzz target", name, variant, flags)
			}
		}
		if checked == 0 {
			t.Errorf("%s has no fuzzer variant on a 64-bit device", name)
		}
	}

	// A fuzzer-free consumer of the same switch still selects ThinLTO.
	thinDep := false
	for _, variant := range ltoVariantsOf(result, "libplaindep") {
		thinDep = thinDep || strings.HasSuffix(variant, "_lto-thin")
	}
	if !thinDep {
		t.Errorf("libplaindep has no lto-thin variant; GLOBAL_THINLTO no longer selects non-fuzz link units")
	}
}
