// Copyright 2022 Gustavo C. Viegas. All rights reserved.

//go:build linux && !android

#include <wayland-client.h>
#include <xdg-shell-client.h>

// openWayland opens the shared library and gets function pointers.
// It is not safe to call any of the C wrappers unless this
// function succeeds.
void* openWayland(void);

// closeWayland closes the shared library.
// It is not safe to call any of the C wrappers after
// calling this function.
void closeWayland(void* handle);

// wl_*_interface.
extern const struct wl_interface displayInterfaceWayland;
extern const struct wl_interface registryInterfaceWayland;
extern const struct wl_interface callbackInterfaceWayland;
extern const struct wl_interface compositorInterfaceWayland;
extern const struct wl_interface shmInterfaceWayland;
extern const struct wl_interface shmPoolInterfaceWayland;
extern const struct wl_interface bufferInterfaceWayland;
extern const struct wl_interface surfaceInterfaceWayland;
extern const struct wl_interface regionInterfaceWayland;
extern const struct wl_interface outputInterfaceWayland;
extern const struct wl_interface seatInterfaceWayland;
extern const struct wl_interface pointerInterfaceWayland;
extern const struct wl_interface keyboardInterfaceWayland;
extern const struct wl_interface touchInterfaceWayland;

// xdg_*_interface.
extern const struct wl_interface wmBaseInterfaceXDG;
extern const struct wl_interface positionerInterfaceXDG;
extern const struct wl_interface surfaceInterfaceXDG;
extern const struct wl_interface toplevelInterfaceXDG;
extern const struct wl_interface popupInterfaceXDG;

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
// - surfaceEnterWayland(sf *C.struct_wl_surface, out *C.struct_wl_output)
// - surfaceLeaveWayland(sf *C.struct_wl_surface, out *C.struct_wl_output)
int surfaceAddListenerWayland(struct wl_surface* sf);

// wl_surface_destroy.
void surfaceDestroyWayland(struct wl_surface* sf);

// xdg_wm_base_add_listener.
// This wrapper requires the following exported Go function:
//
// - wmBasePingXDG(serial C.uint32_t)
int wmBaseAddListenerXDG(struct xdg_wm_base* wm);

// xdg_wm_base_destroy.
void wmBaseDestroyXDG(struct xdg_wm_base* wm);

// xdg_wm_base_create_positioner.
struct xdg_positioner* wmBaseCreatePositionerXDG(struct xdg_wm_base* wm);

// xdg_wm_base_get_xdg_surface.
struct xdg_surface* wmBaseGetXDGSurfaceXDG(struct xdg_wm_base* wm, struct wl_surface* sf);

// xdg_wm_base_pong.
void wmBasePongXDG(struct xdg_wm_base* wm, uint32_t serial);

// wl_seat_add_listener.
// This wrapper requires the following exported Go functions:
//
// - seatCapabilitiesWayland(capab C.uint32_t)
// - seatNameWayland(name *C.char)
int seatAddListenerWayland(struct wl_seat *seat);
