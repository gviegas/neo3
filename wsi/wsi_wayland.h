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

// wl_*_interface.
extern const struct wl_interface registryInterfaceWayland;
extern const struct wl_interface compositorInterfaceWayland;
extern const struct wl_interface surfaceInterfaceWayland;
extern const struct wl_interface regionInterfaceWayland;
extern const struct wl_interface outputInterfaceWayland;
extern const struct wl_interface bufferInterfaceWayland;
extern const struct wl_interface callbackInterfaceWayland;

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
// - registryGlobalWayland(name C.uint32_t, iface *C.char, vers C.uint32_t)
// - registryGlobalRemoveWayland(name C.uint32_t)
int registryAddListenerWayland(struct wl_registry* rty);

// wl_registry_bind.
void* registryBindWayland(struct wl_registry* rty, uint32_t name, const struct wl_interface* iface, uint32_t vers);

// wl_compositor_create_surface.
struct wl_surface* compositorCreateSurfaceWayland(struct wl_compositor* cpt);

// wl_surface_add_listener.
// This wrapper requires the following exported Go functions:
//
// - surfaceEnterWayland(sfc *C.struct_wl_surface, out *C.struct_wl_output)
// - surfaceLeaveWayland(sfc *C.struct_wl_surface, out *C.struct_wl_output)
int surfaceAddListenerWayland(struct wl_surface* sfc);

// wl_surface_destroy.
void surfaceDestroyWayland(struct wl_surface* sfc);
