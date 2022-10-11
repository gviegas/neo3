// Copyright 2022 Gustavo C. Viegas. All rights reserved.

//go:build linux && !android

#include <wayland-client.h>

// openWayland opens the shared library and gets function pointers.
// It is not safe to call any of the C wrappers unless this
// function succeeds.
void* openWayland(void);

// closeWayland closes the shared library.
// It is not safe to call any of the C wrappers after
// calling this function.
void closeWayland(void* handle);

// wl_display_connect.
struct wl_display* displayConnectWayland(const char* name);

// wl_display_disconnect.
void displayDisconnectWayland(struct wl_display* dpy);

// wl_display_dispatch.
int displayDispatchWayland(struct wl_display* dpy);

// wl_display_flush.
int displayFlushWayland(struct wl_display* dpy);

// wl_display_roundtrip.
int displayRoundtripWayland(struct wl_display* dpy);

// wl_display_get_registry.
struct wl_registry* displayGetRegistryWayland(struct wl_display* dpy);

// wl_registry_add_listener.
// This wrapper requires the following exported Go functions:
//
// - func registryGlobalWayland(name C.uint32_t, iface *C.char, vers C.uint32_t)
// - func registryGlobalRemoveWayland(name C.uint32_t)
int registryAddListenerWayland(struct wl_registry* rty);
