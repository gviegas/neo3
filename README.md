# neo3


## Performance

There is a runtime cost associated with calls to foreign code in Go.

We would like to avoid going through CGO when calling "userland" C code. This can be accomplished in assembly by relying on internal details of the Go runtime.
The following code snippet does just that for C's ```rand``` function (amd64):

```c
#include "go_asm.h"
#include "textflag.h"

TEXT ·crand(SB), NOSPLIT, $0
	MOVQ	·rand(SB), AX

	//MOVQ	TLS, R12		// get_tls(R12)
	//MOVQ	0(R12)(TLS*1), R14	// MOVQ	g(R12), R14
	MOVQ	(6*8)(R14), R13		// MOVQ	g_m(R14), R13

	// Switch to g0 stack.
	MOVQ	SP, R12 // Callee-saved; preserved across the CALL.
	MOVQ	(0)(R13), R10		// MOVQ	m_g0(R13), R10
	CMPQ	R10, R14
	JE	call // Already on g0.
	MOVQ	(7*8+0)(R10), SP	// MOVQ	(g_sched+gobuf_sp)(R10), SP
call:
	ANDQ	$~15, SP // Alignment for GCC ABI.
	CALL	AX
	MOVQ	R12, SP
	MOVL	AX, ret+0(FP)
	RET
```

Where ```·rand``` is assumed to store a pointer to the C function. We can obtain such a function pointer from libc:

```go
package caux

// #cgo LDFLAGS: -ldl
// #include <stdlib.h>
// #include <dlfcn.h>
import "C"

import "unsafe"

var dl unsafe.Pointer
var Rand uintptr

func init() {
	cstr := C.CString("libc.so.6")
	defer C.free(unsafe.Pointer(cstr))
	dl = C.dlopen(cstr, C.RTLD_NOW)
	if dl == nil {
		panic("C.dlopen failed")
	}
	cstr = C.CString("rand")
	defer C.free(unsafe.Pointer(cstr))
	sym := C.dlsym(dl, cstr)
	if sym == nil {
		panic("C.dlsym failed")
	}
	Rand = uintptr(sym)
}

func CgoRand() int32 {
	return int32(C.rand())
}
```

Note that the CGO code must be in a different package than the one where we defined the assembly procedure.

Now we can compare ```crand``` with an equivalent function that uses the CGO machinery:

```go
package main

import (
	"testing"

	"go-asm/caux"
)

func crand() int32

var rand = caux.Rand

func BenchmarkAsm(b *testing.B) {
	for b.Loop() {
		crand()
	}
}

func BenchmarkCgo(b *testing.B) {
	for b.Loop() {
		caux.CgoRand()
	}
}
```

Which produces the following output on my computer:

```
goos: linux
goarch: amd64
pkg: go-asm
cpu: Intel(R) Core(TM) i5-4260U CPU @ 1.40GHz
BenchmarkAsm-4   	49187989	        22.13 ns/op
BenchmarkCgo-4   	11163513	       107.4 ns/op
PASS
ok  	go-asm	2.428s
```

Of course, ```rand``` is a simple function that takes no arguments. Passing Go pointers to C code can be particularly problematic.
