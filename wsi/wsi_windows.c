// Copyright 2022 Gustavo C. Viegas. All rights reserved.

#include "_cgo_export.h"

LRESULT CALLBACK wndProcWrapper(HWND hwnd, UINT msg, WPARAM wprm, LPARAM lprm) {
    return wndProcWin32(hwnd, msg, wprm, lprm);
}
