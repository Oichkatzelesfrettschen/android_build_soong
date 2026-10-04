package config

import (
	"strings"
	"testing"
)

// Cortex-A9 implements VFPv3-D16 and NEON without VFPv4/FMA or hardware
// integer divide, so its cflags select -mcpu=cortex-a9 with base NEON and
// never the -mfpu=neon-vfpv4 of the cortex-a7, cortex-a15 and krait rows.
func TestCortexA9CpuVariantCflags(t *testing.T) {
	cflags, ok := armClangCpuVariantCflags["cortex-a9"]
	if !ok {
		t.Fatal("armClangCpuVariantCflags has no \"cortex-a9\" entry")
	}

	joined := strings.Join(cflags, " ")
	if want := "-mcpu=cortex-a9 -mfpu=neon"; joined != want {
		t.Errorf("cortex-a9 cflags = %q, want %q", joined, want)
	}
	if strings.Contains(joined, "vfpv4") {
		t.Errorf("cortex-a9 cflags %q select vfpv4/FMA", joined)
	}
	if strings.Contains(joined, "idiv") {
		t.Errorf("cortex-a9 cflags %q select hardware integer divide", joined)
	}

	varName, ok := armClangCpuVariantCflagsVar["cortex-a9"]
	if !ok {
		t.Fatal("armClangCpuVariantCflagsVar has no \"cortex-a9\" entry")
	}
	if want := "${config.ArmClangCortexA9Cflags}"; varName != want {
		t.Errorf("armClangCpuVariantCflagsVar[\"cortex-a9\"] = %q, want %q", varName, want)
	}
}
