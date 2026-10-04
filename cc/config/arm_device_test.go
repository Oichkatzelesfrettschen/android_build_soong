// Copyright 2026 The Android Open Source Project
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"strings"
	"testing"
)

// Cortex-A9 (e.g. OMAP4460) carries VFPv3-D16/NEON and no VFPv4/FMA unit
// and no hardware integer divide. The cflags this variant emits must
// select -mcpu=cortex-a9 with base NEON, never a -mfpu=neon-vfpv4
// substitute (krait/cortex-a7/cortex-a15's flags), which legally emits
// vfma/vfnm/sdiv/udiv this core does not implement.
func TestCortexA9CpuVariantCflags(t *testing.T) {
	cflags, ok := armClangCpuVariantCflags["cortex-a9"]
	if !ok {
		t.Fatal("armClangCpuVariantCflags has no \"cortex-a9\" entry")
	}

	joined := strings.Join(cflags, " ")
	if joined != "-mcpu=cortex-a9 -mfpu=neon" {
		t.Errorf("cortex-a9 cflags = %q, want %q", joined, "-mcpu=cortex-a9 -mfpu=neon")
	}
	if strings.Contains(joined, "vfpv4") {
		t.Errorf("cortex-a9 cflags %q must not select vfpv4/FMA: this core SIGILLs on vfma/vfnm", joined)
	}
	if strings.Contains(joined, "idiv") {
		t.Errorf("cortex-a9 cflags %q must not select hardware integer divide", joined)
	}

	varName, ok := armClangCpuVariantCflagsVar["cortex-a9"]
	if !ok {
		t.Fatal("armClangCpuVariantCflagsVar has no \"cortex-a9\" entry")
	}
	if varName != "${config.ArmClangCortexA9Cflags}" {
		t.Errorf("armClangCpuVariantCflagsVar[\"cortex-a9\"] = %q, want %q", varName, "${config.ArmClangCortexA9Cflags}")
	}
}
