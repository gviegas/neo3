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

// wl_display_dispatch_pending.
int displayDispatchPendingWayland(struct wl_display* dpy);

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

// wl_registry_destroy.
void registryDestroyWayland(struct wl_registry* rty);

// wl_registry_bind.
void* registryBindWayland(struct wl_registry* rty, uint32_t name, const struct wl_interface* iface, uint32_t vers);

// wl_compositor_destroy.
void compositorDestroyWayland(struct wl_compositor* cpt);

// wl_compositor_create_surface.
struct wl_surface* compositorCreateSurfaceWayland(struct wl_compositor* cpt);

// wl_shm_add_listener.
// This wrapper requires the following exported Go function:
//
// - shmFormatWayland(format C.uint32_t)
int shmAddListenerWayland(struct wl_shm* shm);

// wl_shm_destroy.
void shmDestroyWayland(struct wl_shm* shm);

// wl_shm_create_pool.
struct wl_shm_pool* shmCreatePoolWayland(struct wl_shm* shm, int32_t fd, int32_t size);

// wl_shm_pool_create_buffer.
struct wl_buffer* shmPoolCreateBufferWayland(struct wl_shm_pool* shmp, int32_t offset, int32_t width, int32_t height, int32_t stride, uint32_t format);

// wl_shm_destroy.
void shmPoolDestroyWayland(struct wl_shm_pool* shmp);

// wl_buffer_add_listener.
// This wrapper requires the following exported Go function:
//
// - bufferReleaseWayland(buf *C.struct_wl_buffer)
int bufferAddListenerWayland(struct wl_buffer* buf);

// wl_buffer_destroy.
void bufferDestroyWayland(struct wl_buffer* buf);

// wl_surface_add_listener.
// This wrapper requires the following exported Go functions:
//
// - surfaceEnterWayland(sf *C.struct_wl_surface, out *C.struct_wl_output)
// - surfaceLeaveWayland(sf *C.struct_wl_surface, out *C.struct_wl_output)
int surfaceAddListenerWayland(struct wl_surface* sf);

// wl_surface_destroy.
void surfaceDestroyWayland(struct wl_surface* sf);

// wl_surface_attach.
void surfaceAttachWayland(struct wl_surface* sf, struct wl_buffer* buf, int32_t x, int32_t y);

// wl_surface_frame.
struct wl_callback* surfaceFrameWayland(struct wl_surface* sf);

// wl_surface_commit.
void surfaceCommitWayland(struct wl_surface* sf);

// wl_surface_damage_buffer.
void surfaceDamageBufferWayland(struct wl_surface* sf, int32_t x, int32_t y, int32_t width, int32_t height);

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

// xdg_positioner_destroy.
void positionerDestroyXDG(struct xdg_positioner* pos);

// xdg_surface_add_listener.
// This wrapper requires the following exported Go function:
//
// - surfaceConfigureXDG(xsf *C.struct_xdg_surface, serial C.uint32_t)
int surfaceAddListenerXDG(struct xdg_surface* xsf);

// xdg_surface_destroy.
void surfaceDestroyXDG(struct xdg_surface* xsf);

// xdg_surface_get_toplevel.
struct xdg_toplevel* surfaceGetToplevelXDG(struct xdg_surface* xsf);

// xdg_surface_get_popup.
struct xdg_popup* surfaceGetPopupXDG(struct xdg_surface* xsf, struct xdg_surface* parent, struct xdg_positioner* pos);

// xdg_surface_set_window_geometry.
void surfaceSetWindowGeometryXDG(struct xdg_surface* xsf, int32_t x, int32_t y, int32_t width, int32_t height);

// xdg_surface_ack_configure.
void surfaceAckConfigureXDG(struct xdg_surface* xsf, uint32_t serial);

// xdg_toplevel_add_listener.
// This wrapper requires the following exported Go functions:
//
// - toplevelConfigureXDG(tl *C.struct_xdg_toplevel, width, height C.int32_t, states *C.struct_wl_array)
// - toplevelCloseXDG(tl *C.struct_xdg_toplevel)
// - toplevelConfigureBoundsXDG(tl *C.struct_xdg_toplevel, width, height C.int32_t)
int toplevelAddListenerXDG(struct xdg_toplevel* tl);

// xdg_toplevel_destroy.
void toplevelDestroyXDG(struct xdg_toplevel* tl);

// xdg_toplevel_set_parent.
void toplevelSetParentXDG(struct xdg_toplevel* tl, struct xdg_toplevel* parent);

// xdg_toplevel_set_title.
void toplevelSetTitleXDG(struct xdg_toplevel* tl, const char* title);

// xdg_toplevel_set_app_id.
void toplevelSetAppIDXDG(struct xdg_toplevel* tl, const char* appID);

// xdg_toplevel_set_max_size.
void toplevelSetMaxSizeXDG(struct xdg_toplevel* tl, int32_t width, int32_t height);

// xdg_toplevel_set_min_size.
void toplevelSetMinSizeXDG(struct xdg_toplevel* tl, int32_t width, int32_t height);

// xdg_toplevel_set_fullscreen.
void toplevelSetFullscreenXDG(struct xdg_toplevel* tl, struct wl_output* out);

// xdg_toplevel_unset_fullscreen.
void toplevelUnsetFullscreenXDG(struct xdg_toplevel* tl);

// wl_seat_add_listener.
// This wrapper requires the following exported Go functions:
//
// - seatCapabilitiesWayland(capab C.uint32_t)
// - seatNameWayland(name *C.char)
int seatAddListenerWayland(struct wl_seat *seat);

// wl_seat_destroy.
void seatDestroyWayland(struct wl_seat* seat);

// wl_seat_get_pointer.
struct wl_pointer* seatGetPointerWayland(struct wl_seat* seat);

// wl_seat_get_keyboard.
struct wl_keyboard* seatGetKeyboardWayland(struct wl_seat* seat);

// wl_seat_release.
void seatReleaseWayland(struct wl_seat* seat);

// wl_pointer_add_listener.
// This wrapper requires the following exported Go functions:
//
// - pointerEnterWayland(serial C.uint32_t, sf *C.struct_wl_surface, x, y C.wl_fixed_t)
// - pointerLeaveWayland(serial C.uint32_t, sf *C.struct_wl_surface)
// - pointerMotionWayland(millis C.uint32_t, x, y C.wl_fixed_t)
// - pointerButtonWayland(serial, millis, button, state C.uint32_t)
// - pointerAxisWayland(millis, axis C.uint32_t, value C.wl_fixed_t)
// - pointerFrameWayland()
// - pointerAxisSourceWayland(axisSrc C.uint32_t)
// - pointerAxisStopWayland(millis, axis C.uint32_t)
// - pointerAxisDiscreteWayland(axis C.uint32_t, discrete C.int32_t)
int pointerAddListenerWayland(struct wl_pointer* pt);

// wl_pointer_destroy.
void pointerDestroyWayland(struct wl_pointer* pt);

// wl_pointer_set_cursor.
void pointerSetCursorWayland(struct wl_pointer* pt, uint32_t serial, struct wl_surface* sf, int32_t hotspotX, int32_t hotspotY);

// wl_pointer_release.
void pointerReleaseWayland(struct wl_pointer* pt);

// wl_keyboard_add_listener.
// This wrapper requires the following exported Go functions:
//
// - keyboardKeymapWayland(format C.uint32_t, fd C.int32_t, size C.uint32_t)
// - keyboardEnterWayland(serial C.uint32_t, sf *C.struct_wl_surface, keys *C.struct_wl_array)
// - keyboardLeaveWayland(serial C.uint32_t, sf *C.struct_wl_surface)
// - keyboardKeyWayland(serial, millis, key, state C.uint32_t)
// - keyboardModifiersWayland(serial, depressed, latched, locked, group C.uint32_t)
// - keyboardRepeatInfoWayland(rate, delay C.int32_t)
int keyboardAddListenerWayland(struct wl_keyboard* kb);

// wl_keyboard_destroy.
void keyboardDestroyWayland(struct wl_keyboard* kb);

// wl_keyboard_release.
void keyboardReleaseWayland(struct wl_keyboard* kb);
