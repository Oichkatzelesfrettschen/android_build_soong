package config

import (
	"reflect"
	"strings"
	"testing"

	"android/soong/android"
)

func clangTestConfig(releaseVersion string) android.Config {
	env := map[string]string{}
	if releaseVersion != "" {
		env["LLVM_RELEASE_VERSION"] = releaseVersion
	}
	return android.TestConfig("", env, "", nil)
}

func TestParseClangMajorVersion(t *testing.T) {
	testCases := []struct {
		version string
		major   int
		valid   bool
	}{
		{"11.0.2", 11, true},
		{"22", 22, true},
		{"22.0.0", 22, true},
		{"", 0, false},
		{"r584948", 0, false},
	}
	for _, tc := range testCases {
		major, err := parseClangMajorVersion(tc.version)
		if tc.valid && (err != nil || major != tc.major) {
			t.Errorf("parseClangMajorVersion(%q) = %d, %v; want %d", tc.version, major, err, tc.major)
		}
		if !tc.valid && err == nil {
			t.Errorf("parseClangMajorVersion(%q) = %d; want an error", tc.version, major)
		}
	}
}

func TestClangMajorVersion(t *testing.T) {
	if got := ClangMajorVersion(clangTestConfig("")); got != 11 {
		t.Errorf("ClangMajorVersion with LLVM_RELEASE_VERSION unset = %d, want 11", got)
	}
	if got := ClangMajorVersion(clangTestConfig("22")); got != 22 {
		t.Errorf("ClangMajorVersion with LLVM_RELEASE_VERSION=22 = %d, want 22", got)
	}
}

// The Clang 11 strings are the values the default compiler receives; any
// change to them changes every compile command of a Clang 11 build.
func TestClang11FlagStrings(t *testing.T) {
	cfg := clangTestConfig("")

	wantExtra := "-D__compiler_offsetof=__builtin_offsetof -faddrsig -Werror=int-conversion " +
		"-fexperimental-new-pass-manager -Wno-reserved-id-macro -Wno-unused-command-line-argument " +
		"-fcolor-diagnostics -Wno-sign-compare -Wno-defaulted-function-deleted " +
		"-Wno-inconsistent-missing-override -Wno-c99-designator"
	if got := strings.Join(clangExtraCflags(cfg), " "); got != wantExtra {
		t.Errorf("ClangExtraCflags = %q, want %q", got, wantExtra)
	}

	wantNoOverride := "-Werror=address-of-temporary -Werror=return-type " +
		"-Wno-tautological-constant-compare -Wno-tautological-type-limit-compare " +
		"-Wno-reorder-init-list -Wno-implicit-int-float-conversion -Wno-int-in-bool-context " +
		"-Wno-sizeof-array-div -Wno-tautological-overlap-compare -Wno-deprecated-copy " +
		"-Wno-range-loop-construct -Wno-misleading-indentation -Wno-zero-as-null-pointer-constant " +
		"-Wno-deprecated-anon-enum-enum-conversion -Wno-deprecated-enum-enum-conversion " +
		"-Wno-string-compare -Wno-enum-enum-conversion -Wno-enum-float-conversion " +
		"-Wno-pessimizing-move"
	if got := strings.Join(clangExtraNoOverrideCflags(cfg), " "); got != wantNoOverride {
		t.Errorf("ClangExtraNoOverrideCflags = %q, want %q", got, wantNoOverride)
	}

	wantCommon := strings.Join(append(ClangFilterUnknownCflags(commonGlobalCflags),
		"${ClangExtraCflags}",
		"-ftrivial-auto-var-init=zero -enable-trivial-auto-var-init-zero-knowing-it-will-be-removed-from-clang"), " ")
	if got := commonClangGlobalCflags(cfg); got != wantCommon {
		t.Errorf("CommonClangGlobalCflags = %q, want %q", got, wantCommon)
	}

	sanitizerCflags := []string{"-fno-omit-frame-pointer", "-fno-experimental-new-pass-manager"}
	if got := ClangFilterPassManagerCflags(cfg, sanitizerCflags); !reflect.DeepEqual(got, sanitizerCflags) {
		t.Errorf("ClangFilterPassManagerCflags = %q, want %q", got, sanitizerCflags)
	}
	if !ClangHasLegacyPassManager(cfg) {
		t.Error("ClangHasLegacyPassManager = false, want true")
	}
	if got := TidyChecksDisabledForClang(cfg); len(got) != 0 {
		t.Errorf("TidyChecksDisabledForClang = %q, want none", got)
	}
	if got := LinuxGlibcClangCppflags(cfg); len(got) != 0 {
		t.Errorf("LinuxGlibcClangCppflags = %q, want none", got)
	}
}

func TestClang14PassManagerSpelling(t *testing.T) {
	cfg := clangTestConfig("14")
	got := ClangFilterPassManagerCflags(cfg, []string{"-a", "-fexperimental-new-pass-manager",
		"-fno-experimental-new-pass-manager", "-b"})
	want := []string{"-a", "-flegacy-pass-manager", "-b"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ClangFilterPassManagerCflags with Clang 14 = %q, want %q", got, want)
	}
	if got := ClangFilterPassManagerCflags(clangTestConfig("22"),
		[]string{"-fno-experimental-new-pass-manager"}); len(got) != 0 {
		t.Errorf("ClangFilterPassManagerCflags with Clang 22 = %q, want none", got)
	}
}

func TestClang22FlagStrings(t *testing.T) {
	cfg := clangTestConfig("22")

	for _, f := range clangExtraCflags(cfg) {
		if f == "-fexperimental-new-pass-manager" || f == "-fno-experimental-new-pass-manager" {
			t.Errorf("ClangExtraCflags carries %q, which Clang 22 rejects", f)
		}
	}

	common := commonClangGlobalCflags(cfg)
	if strings.Contains(common, "-enable-trivial-auto-var-init-zero-knowing-it-will-be-removed-from-clang") {
		t.Errorf("CommonClangGlobalCflags %q carries the zero-init opt-in flag Clang 22 rejects", common)
	}
	if !strings.HasSuffix(common, " -ftrivial-auto-var-init=zero") {
		t.Errorf("CommonClangGlobalCflags %q does not end with -ftrivial-auto-var-init=zero", common)
	}

	noOverride := clangExtraNoOverrideCflags(cfg)
	clang11NoOverride := clangExtraNoOverrideCflags(clangTestConfig(""))
	if !reflect.DeepEqual(noOverride, append(append([]string{}, clang11NoOverride...), clang22NoOverrideCflags...)) {
		t.Errorf("ClangExtraNoOverrideCflags = %q, want the Clang 11 list followed by %q", noOverride, clang22NoOverrideCflags)
	}

	sanitizerCflags := []string{"-fno-omit-frame-pointer", "-fno-experimental-new-pass-manager", "-mllvm"}
	if got, want := ClangFilterPassManagerCflags(cfg, sanitizerCflags), []string{"-fno-omit-frame-pointer", "-mllvm"}; !reflect.DeepEqual(got, want) {
		t.Errorf("ClangFilterPassManagerCflags = %q, want %q", got, want)
	}
	if ClangHasLegacyPassManager(cfg) {
		t.Error("ClangHasLegacyPassManager = true, want false")
	}

	gotTidy := TidyChecksDisabledForClang(cfg)
	if len(gotTidy) != len(clangTidy22NewChecks) {
		t.Errorf("TidyChecksDisabledForClang has %d exclusions, want %d", len(gotTidy), len(clangTidy22NewChecks))
	}
	for _, want := range []string{"-misc-const-correctness", "-misc-include-cleaner",
		"-bugprone-easily-swappable-parameters", "-cert-int09-c", "-cert-err33-c",
		"-misc-use-anonymous-namespace", "-clang-analyzer-security.ArrayBound"} {
		if !inList(want, gotTidy) {
			t.Errorf("TidyChecksDisabledForClang lacks %q", want)
		}
	}
	if got, want := LinuxGlibcClangCppflags(cfg), []string{"-isystem build/soong/cc/config/hostcxx"}; !reflect.DeepEqual(got, want) {
		t.Errorf("LinuxGlibcClangCppflags = %q, want %q", got, want)
	}
}
