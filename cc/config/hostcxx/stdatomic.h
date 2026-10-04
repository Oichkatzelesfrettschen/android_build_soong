/* SPDX-License-Identifier: Apache-2.0 */
/*
 * Host C++ finds <stdatomic.h> here, ahead of the compiler's resource
 * header. Clang 22's resource <stdatomic.h> defines the C11 atomic macros in
 * C++ mode, which breaks libc++'s <atomic>, and the glibc host sysroot has
 * no <stdatomic.h> of its own. Bionic's header maps the C11 names onto
 * libc++'s <atomic> under C++, as device code already sees it.
 */
#include "../../../../../bionic/libc/include/stdatomic.h"
