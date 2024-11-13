// Copyright 2022 Gustavo C. Viegas. All rights reserved.

//go:build linux && !android

#include <dlfcn.h>
#include <stdlib.h>
#include <string.h>
#include <wsi_wayland.h>
#include <_cgo_export.h>

#define LIBWAYLAND "libwayland-client.so.0"

static struct wl_display* (*displayConnect)(const char*);
static void (*displayDisconnect)(struct wl_display*);
static int (*displayDispatch)(struct wl_display*);
static int (*displayDispatchPending)(struct wl_display*);
static int (*displayFlush)(struct wl_display*);
static int (*displayRoundtrip)(struct wl_display*);
static void (*proxyDestroy)(struct wl_proxy*);
static int (*proxyAddListener)(struct wl_proxy*, void (**)(void), void*);
static uint32_t (*proxyGetVersion)(struct wl_proxy*);
static struct wl_proxy* (*proxyMarshalFlags)(struct wl_proxy*, uint32_t, const struct wl_interface*, uint32_t, uint32_t, ...);

void* openWayland(void) {
	void* handle = dlopen(LIBWAYLAND, RTLD_LAZY|RTLD_LOCAL);
	if (handle == NULL)
		return NULL;

	displayConnect = dlsym(handle, "wl_display_connect");
	if (displayConnect == NULL)
		goto nosym;
	displayDisconnect = dlsym(handle, "wl_display_disconnect");
	if (displayDisconnect == NULL)
		goto nosym;
	displayDispatch = dlsym(handle, "wl_display_dispatch");
	if (displayDispatch == NULL)
		goto nosym;
	displayDispatchPending = dlsym(handle, "wl_display_dispatch_pending");
	if (displayDispatchPending == NULL)
		goto nosym;
	displayFlush = dlsym(handle, "wl_display_flush");
	if (displayFlush == NULL)
		goto nosym;
	displayRoundtrip = dlsym(handle, "wl_display_roundtrip");
	if (displayRoundtrip == NULL)
		goto nosym;
	proxyDestroy = dlsym(handle, "wl_proxy_destroy");
	if (proxyDestroy == NULL)
		goto nosym;
	proxyAddListener = dlsym(handle, "wl_proxy_add_listener");
	if (proxyAddListener == NULL)
		goto nosym;
	proxyGetVersion = dlsym(handle, "wl_proxy_get_version");
	if (proxyGetVersion == NULL)
		goto nosym;
	proxyMarshalFlags = dlsym(handle, "wl_proxy_marshal_flags");
	if (proxyMarshalFlags == NULL)
		goto nosym;

	return handle;

nosym:
	dlclose(handle);
	return NULL;
}

void closeWayland(void* handle) {
	dlclose(handle);
}

static const struct wl_interface* nullInterface[8];

// TODO
const struct wl_interface displayInterfaceWayland;

const struct wl_interface registryInterfaceWayland = {
	.name = "wl_registry",
	.version = 1,
	.method_count = 1,
	.methods = (const struct wl_message[1]){
		{ "bind", "usun", nullInterface }
	},
	.event_count = 2,
	.events = (const struct wl_message[2]){
		{ "global", "usu", nullInterface },
		{ "global_remove", "u", nullInterface },
	},
};

const struct wl_interface callbackInterfaceWayland = {
	.name = "wl_callback",
	.version = 1,
	.method_count = 0,
	.methods = NULL,
	.event_count = 1,
	.events = (const struct wl_message[1]){
		{ "done", "u", nullInterface },
	},
};

const struct wl_interface compositorInterfaceWayland = {
	.name = "wl_compositor",
	.version = 6,
	.method_count = 2,
	.methods = (const struct wl_message[2]){
		{ "create_surface", "n", (const struct wl_interface*[1]){&surfaceInterfaceWayland} },
		{ "create_region", "n", (const struct wl_interface*[1]){&regionInterfaceWayland} },
	},
	.event_count = 0,
	.events = NULL,
};

const struct wl_interface shmInterfaceWayland = {
	.name = "wl_shm",
	.version = 2,
	.method_count = 2,
	.methods = (const struct wl_message[2]){
		{ "create_pool", "nhi", (const struct wl_interface*[3]){&shmPoolInterfaceWayland} },
		{ "release", "2", nullInterface },
	},
	.event_count = 1,
	.events = (const struct wl_message[1]){
		{ "format", "u", nullInterface },
	},
};

const struct wl_interface shmPoolInterfaceWayland = {
	.name = "wl_shm_pool",
	.version = 2,
	.method_count = 3,
	.methods = (const struct wl_message[3]){
		{ "create_buffer", "niiiiu", (const struct wl_interface*[6]){&bufferInterfaceWayland} },
		{ "destroy", "", nullInterface },
		{ "resize", "i", nullInterface },
	},
	.event_count = 0,
	.events = NULL,
};

const struct wl_interface bufferInterfaceWayland = {
	.name = "wl_buffer",
	.version = 1,
	.method_count = 1,
	.methods = (const struct wl_message[1]){
		{ "destroy", "", nullInterface },
	},
	.event_count = 1,
	.events = (const struct wl_message[1]){
		{ "release", "", nullInterface },
	},
};

const struct wl_interface surfaceInterfaceWayland = {
	.name = "wl_surface",
	.version = 6,
	.method_count = 11,
	.methods = (const struct wl_message[11]){
		{ "destroy", "", nullInterface },
		{ "attach", "?oii", (const struct wl_interface*[3]){&bufferInterfaceWayland} },
		{ "damage", "iiii", nullInterface },
		{ "frame", "n", (const struct wl_interface*[1]){&callbackInterfaceWayland} },
		{ "set_opaque_region", "?o", (const struct wl_interface*[1]){&regionInterfaceWayland} },
		{ "set_input_region", "?o", (const struct wl_interface*[1]){&regionInterfaceWayland} },
		{ "commit", "", nullInterface },
		{ "set_buffer_transform", "2i", nullInterface },
		{ "set_buffer_scale", "3i", nullInterface },
		{ "damage_buffer", "4iiii", nullInterface },
		{ "offset", "5ii", nullInterface },
	},
	.event_count = 4,
	.events = (const struct wl_message[4]){
		{ "enter", "o", (const struct wl_interface*[1]){&outputInterfaceWayland} },
		{ "leave", "o", (const struct wl_interface*[1]){&outputInterfaceWayland} },
		{ "preferred_buffer_scale", "6i", nullInterface },
		{ "preferred_buffer_transform", "6u", nullInterface },
	},
};

const struct wl_interface regionInterfaceWayland = {
	.name = "wl_region",
	.version = 1,
	.method_count = 3,
	.methods = (const struct wl_message[3]){
		{ "destroy", "", nullInterface },
		{ "add", "iiii", nullInterface },
		{ "subtract", "iiii", nullInterface },
	},
	.event_count = 0,
	.events = NULL,
};

const struct wl_interface outputInterfaceWayland = {
	.name = "wl_output",
	.version = 4,
	.method_count = 1,
	.methods = (const struct wl_message[1]){
		{ "release", "3", nullInterface },
	},
	.event_count = 6,
	.events = (const struct wl_message[6]){
		{ "geometry", "iiiiissi", nullInterface },
		{ "mode", "uiii", nullInterface },
		{ "done", "2", nullInterface },
		{ "scale", "2i", nullInterface },
		{ "name", "4s", nullInterface },
		{ "description", "4s", nullInterface },
	},
};

const struct wl_interface seatInterfaceWayland = {
	.name = "wl_seat",
	.version = 9,
	.method_count = 4,
	.methods = (const struct wl_message[4]){
		{ "get_pointer", "n", (const struct wl_interface*[1]){&pointerInterfaceWayland} },
		{ "get_keyboard", "n", (const struct wl_interface*[1]){&keyboardInterfaceWayland} },
		{ "get_touch", "n", (const struct wl_interface*[1]){&touchInterfaceWayland} },
		{ "release", "5", nullInterface },
	},
	.event_count = 2,
	.events = (const struct wl_message[2]){
		{ "capabilities", "u", nullInterface },
		{ "name", "2s", nullInterface },
	},
};

const struct wl_interface pointerInterfaceWayland = {
	.name = "wl_pointer",
	.version = 7,
	.method_count = 2,
	.methods = (const struct wl_message[2]){
		{ "set_cursor", "u?oii", (const struct wl_interface*[4]){NULL, &surfaceInterfaceWayland} },
		{ "release", "3", nullInterface },
	},
	.event_count = 9,
	.events = (const struct wl_message[9]){
		{ "enter", "uoff", (const struct wl_interface*[4]){NULL, &surfaceInterfaceWayland} },
		{ "leave", "uo", (const struct wl_interface*[2]){NULL, &surfaceInterfaceWayland} },
		{ "motion", "uff", nullInterface },
		{ "button", "uuuu", nullInterface },
		{ "axis", "uuf", nullInterface },
		{ "frame", "5", nullInterface },
		{ "axis_source", "5u", nullInterface },
		{ "axis_stop", "5uu", nullInterface },
		{ "axis_discrete", "5ui", nullInterface },
	},
};

const struct wl_interface keyboardInterfaceWayland = {
	.name = "wl_keyboard",
	.version = 9,
	.method_count = 1,
	.methods = (const struct wl_message[1]){
		{ "release", "3", nullInterface },
	},
	.event_count = 6,
	.events = (const struct wl_message[6]){
		{ "keymap", "uhu", nullInterface },
		{ "enter", "uoa", (const struct wl_interface*[3]){NULL, &surfaceInterfaceWayland} },
		{ "leave", "uo", (const struct wl_interface*[2]){NULL, &surfaceInterfaceWayland} },
		{ "key", "uuuu", nullInterface },
		{ "modifiers", "uuuuu", nullInterface },
		{ "repeat_info", "4ii", nullInterface },
	},
};

const struct wl_interface touchInterfaceWayland = {
	.name = "wl_touch",
	.version = 9,
	.method_count = 1,
	.methods = (const struct wl_message[1]){
		{ "release", "3", nullInterface },
	},
	.event_count = 7,
	.events = (const struct wl_message[7]){
		{ "down", "uuoiff", (const struct wl_interface*[6]){NULL, NULL, &surfaceInterfaceWayland} },
		{ "up", "uui", nullInterface },
		{ "motion", "uiff", nullInterface },
		{ "frame", "", nullInterface },
		{ "cancel", "", nullInterface },
		{ "shape", "6iff", nullInterface },
		{ "orientation", "6if", nullInterface },
	},
};

const struct wl_interface wmBaseInterfaceXDG = {
	.name = "xdg_wm_base",
	.version = 4,
	.method_count = 4,
	.methods = (const struct wl_message[4]){
		{ "destroy", "", nullInterface },
		{ "create_positioner", "n", (const struct wl_interface*[1]){&positionerInterfaceXDG} },
		{ "get_xdg_surface", "no", (const struct wl_interface*[2]){&surfaceInterfaceXDG, &surfaceInterfaceWayland} },
		{ "pong", "u", nullInterface },
	},
	.event_count = 1,
	.events = (const struct wl_message[1]){
		{ "ping", "u", nullInterface },
	},
};

const struct wl_interface positionerInterfaceXDG = {
	.name = "xdg_positioner",
	.version = 4,
	.method_count = 10,
	.methods = (const struct wl_message[10]){
		{ "destroy", "", nullInterface },
		{ "set_size", "ii", nullInterface },
		{ "set_anchor_rect", "iiii", nullInterface },
		{ "set_anchor", "u", nullInterface },
		{ "set_gravity", "u", nullInterface },
		{ "set_constraint_adjustment", "u", nullInterface },
		{ "set_offset", "ii", nullInterface },
		{ "set_reactive", "3", nullInterface },
		{ "set_parent_size", "3ii", nullInterface },
		{ "set_parent_configure", "3u", nullInterface },
	},
	.event_count = 0,
	.events = NULL,
};

const struct wl_interface surfaceInterfaceXDG = {
	.name = "xdg_surface",
	.version = 4,
	.method_count = 5,
	.methods = (const struct wl_message[5]){
		{ "destroy", "", nullInterface },
		{ "get_toplevel", "n", (const struct wl_interface*[1]){&toplevelInterfaceXDG} },
		{ "get_popup", "n?oo", (const struct wl_interface*[3]){&popupInterfaceXDG, &surfaceInterfaceXDG, &positionerInterfaceXDG} },
		{ "set_window_geometry", "iiii", nullInterface },
		{ "ack_configure", "u", nullInterface },
	},
	.event_count = 1,
	.events = (const struct wl_message[1]) {
		{ "configure", "u", nullInterface },
	},
};

const struct wl_interface toplevelInterfaceXDG = {
	.name = "xdg_toplevel",
	.version = 6,
	.method_count = 14,
	.methods = (const struct wl_message[14]) {
		{ "destroy", "", nullInterface },
		{ "set_parent", "?o", (const struct wl_interface*[1]){&toplevelInterfaceXDG} },
		{ "set_title", "s", nullInterface },
		{ "set_app_id", "s", nullInterface },
		{ "show_window_menu", "ouii", (const struct wl_interface*[4]){&seatInterfaceWayland} },
		{ "move", "ou", (const struct wl_interface*[2]){&seatInterfaceWayland} },
		{ "resize", "ouu", (const struct wl_interface*[3]){&seatInterfaceWayland} },
		{ "set_max_size", "ii", nullInterface },
		{ "set_min_size", "ii", nullInterface },
		{ "set_maximized", "", nullInterface },
		{ "unset_maximized", "", nullInterface },
		{ "set_fullscreen", "?o", (const struct wl_interface*[1]){&outputInterfaceWayland} },
		{ "unset_fullscreen", "", nullInterface },
		{ "set_minimized", "", nullInterface },
	},
	.event_count = 4,
	.events = (const struct wl_message[4]) {
		{ "configure", "iia", nullInterface },
		{ "close", "", nullInterface },
		{ "configure_bounds", "4ii", nullInterface },
		{ "wm_capabilities", "5a", nullInterface },
	},
};

const struct wl_interface popupInterfaceXDG = {
	.name = "xdg_popup",
	.version = 4,
	.method_count = 3,
	.methods = (const struct wl_message[3]) {
		{ "destroy", "", nullInterface },
		{ "grab", "ou", (const struct wl_interface*[2]){&seatInterfaceWayland} },
		{ "reposition", "3ou", (const struct wl_interface*[2]){&positionerInterfaceXDG} },
	},
	.event_count = 3,
	.events = (const struct wl_message[3]) {
		{ "configure", "iiii", nullInterface },
		{ "popup_done", "", nullInterface },
		{ "repositioned", "3u", nullInterface },
	},
};

struct wl_display* displayConnectWayland(const char* name) {
	return displayConnect(name);
}

void displayDisconnectWayland(struct wl_display* dpy) {
	displayDisconnect(dpy);
}

int displayDispatchWayland(struct wl_display* dpy) {
	return displayDispatch(dpy);
}

int displayDispatchPendingWayland(struct wl_display* dpy) {
	return displayDispatchPending(dpy);
}

int displayFlushWayland(struct wl_display* dpy) {
	return displayFlush(dpy);
}

int displayRoundtripWayland(struct wl_display* dpy) {
	return displayRoundtrip(dpy);
}

struct wl_registry* displayGetRegistryWayland(struct wl_display* dpy) {
	return (struct wl_registry*)proxyMarshalFlags(
		(struct wl_proxy*)dpy, WL_DISPLAY_GET_REGISTRY, &registryInterfaceWayland, proxyGetVersion((struct wl_proxy*)dpy), 0, NULL);
}

static void registryGlobal(void* data_, struct wl_registry* rty_, uint32_t name, const char* iface, uint32_t vers) {
	char* s = strdup(iface);
	if (s == NULL)
		return;
	registryGlobalWayland(name, s, vers);
	free(s);
}

static void registryGlobalRemove(void* data_, struct wl_registry* rty_, uint32_t name) {
	registryGlobalRemoveWayland(name);
}

int registryAddListenerWayland(struct wl_registry* rty) {
	static const struct wl_registry_listener ltn = {
		.global = registryGlobal,
		.global_remove = registryGlobalRemove,
	};
	return proxyAddListener((struct wl_proxy*)rty, (void (**)(void))&ltn, NULL);
}

void registryDestroyWayland(struct wl_registry* rty) {
	proxyDestroy((struct wl_proxy*)rty);
}

void* registryBindWayland(struct wl_registry* rty, uint32_t name, const struct wl_interface* iface, uint32_t vers) {
      return proxyMarshalFlags((struct wl_proxy*)rty, WL_REGISTRY_BIND, iface, vers, 0, name, iface->name, vers, NULL);
}

void compositorDestroyWayland(struct wl_compositor* cpt) {
	proxyDestroy((struct wl_proxy*)cpt);
}

struct wl_surface* compositorCreateSurfaceWayland(struct wl_compositor* cpt) {
	return (struct wl_surface*)proxyMarshalFlags(
		(struct wl_proxy*)cpt, WL_COMPOSITOR_CREATE_SURFACE, &surfaceInterfaceWayland, proxyGetVersion((struct wl_proxy*)cpt), 0, NULL);
}

static void shmFormat(void* data_, struct wl_shm* shm_, uint32_t format) {
	shmFormatWayland(format);
}

int shmAddListenerWayland(struct wl_shm* shm) {
	static const struct wl_shm_listener ltn = { shmFormat };
	return proxyAddListener((struct wl_proxy*)shm, (void (**)(void))&ltn, NULL);
}

void shmDestroyWayland(struct wl_shm* shm) {
	proxyDestroy((struct wl_proxy*)shm);
}

struct wl_shm_pool* shmCreatePoolWayland(struct wl_shm* shm, int32_t fd, int32_t size) {
	return (struct wl_shm_pool*)proxyMarshalFlags(
		(struct wl_proxy*)shm, WL_SHM_CREATE_POOL, &shmPoolInterfaceWayland, proxyGetVersion((struct wl_proxy*)shm), 0, NULL, fd, size);
}

void shmReleaseWayland(struct wl_shm* shm) {
	proxyMarshalFlags((struct wl_proxy*)shm, WL_SHM_RELEASE, NULL, proxyGetVersion((struct wl_proxy*)shm), WL_MARSHAL_FLAG_DESTROY);
}

struct wl_buffer* shmPoolCreateBufferWayland(struct wl_shm_pool* shmp, int32_t offset, int32_t width, int32_t height, int32_t stride, uint32_t format) {
	return (struct wl_buffer*)proxyMarshalFlags(
		(struct wl_proxy*)shmp, WL_SHM_POOL_CREATE_BUFFER, &bufferInterfaceWayland, proxyGetVersion((struct wl_proxy*)shmp), 0, NULL, offset, width, height, stride, format);
}

void shmPoolDestroyWayland(struct wl_shm_pool* shmp) {
	proxyMarshalFlags((struct wl_proxy*)shmp, WL_SHM_POOL_DESTROY, NULL, proxyGetVersion((struct wl_proxy*)shmp), WL_MARSHAL_FLAG_DESTROY);
}

static void bufferRelease(void* data_, struct wl_buffer* buf) {
	bufferReleaseWayland(buf);
}

int bufferAddListenerWayland(struct wl_buffer* buf) {
	static struct wl_buffer_listener ltn = { bufferRelease };
	return proxyAddListener((struct wl_proxy*)buf, (void (**)(void))&ltn, NULL);
}

void bufferDestroyWayland(struct wl_buffer* buf) {
	proxyMarshalFlags((struct wl_proxy*)buf, WL_BUFFER_DESTROY, NULL, proxyGetVersion((struct wl_proxy*)buf), WL_MARSHAL_FLAG_DESTROY);
}

static void surfaceEnter(void* data_, struct wl_surface* sf, struct wl_output* out) {
	surfaceEnterWayland(sf, out);
}

static void surfaceLeave(void* data_, struct wl_surface* sf, struct wl_output* out) {
	surfaceLeaveWayland(sf, out);
}

static void surfacePreferredBufferScale(void* data_, struct wl_surface* sf, int32_t factor) {
	surfacePreferredBufferScaleWayland(sf, factor);
}

static void surfacePreferredBufferTransform(void* data_, struct wl_surface* sf, uint32_t xform) {
	surfacePreferredBufferTransformWayland(sf, xform);
}

int surfaceAddListenerWayland(struct wl_surface* sf) {
	static const struct wl_surface_listener ltn = {
		.enter = surfaceEnter,
		.leave = surfaceLeave,
		.preferred_buffer_scale = surfacePreferredBufferScale,
		.preferred_buffer_transform = surfacePreferredBufferTransform,
	};
	return proxyAddListener((struct wl_proxy*)sf, (void (**)(void))&ltn, NULL);
}

void surfaceDestroyWayland(struct wl_surface* sf) {
	proxyMarshalFlags((struct wl_proxy*)sf, WL_SURFACE_DESTROY, NULL, proxyGetVersion((struct wl_proxy*)sf), WL_MARSHAL_FLAG_DESTROY);
}

void surfaceAttachWayland(struct wl_surface* sf, struct wl_buffer* buf, int32_t x, int32_t y) {
	proxyMarshalFlags((struct wl_proxy*)sf, WL_SURFACE_ATTACH, NULL, proxyGetVersion((struct wl_proxy*)sf), 0, buf, x, y);
}

struct wl_callback* surfaceFrameWayland(struct wl_surface* sf) {
	return (struct wl_callback*)proxyMarshalFlags(
		(struct wl_proxy*)sf, WL_SURFACE_FRAME, &callbackInterfaceWayland, proxyGetVersion((struct wl_proxy*)sf), 0, NULL);
}

void surfaceCommitWayland(struct wl_surface* sf) {
	proxyMarshalFlags((struct wl_proxy*)sf, WL_SURFACE_COMMIT, NULL, proxyGetVersion((struct wl_proxy*)sf), 0);
}

void surfaceDamageBufferWayland(struct wl_surface* sf, int32_t x, int32_t y, int32_t width, int32_t height) {
	proxyMarshalFlags((struct wl_proxy*)sf, WL_SURFACE_DAMAGE_BUFFER, NULL, proxyGetVersion((struct wl_proxy*)sf), 0, x, y, width, height);
}

static void wmBasePing(void* data_, struct xdg_wm_base* wm_, uint32_t serial) {
	wmBasePingXDG(serial);
}

int wmBaseAddListenerXDG(struct xdg_wm_base* wm) {
	static const struct xdg_wm_base_listener ltn = { wmBasePing };
	return proxyAddListener((struct wl_proxy*)wm, (void (**)(void))&ltn, NULL);
}

void wmBaseDestroyXDG(struct xdg_wm_base* wm) {
	proxyMarshalFlags((struct wl_proxy*)wm, XDG_WM_BASE_DESTROY, NULL, proxyGetVersion((struct wl_proxy*)wm), WL_MARSHAL_FLAG_DESTROY);
}

struct xdg_positioner* wmBaseCreatePositionerXDG(struct xdg_wm_base* wm) {
	return (struct xdg_positioner*)proxyMarshalFlags(
		(struct wl_proxy*)wm, XDG_WM_BASE_CREATE_POSITIONER, &positionerInterfaceXDG, proxyGetVersion((struct wl_proxy*)wm), 0, NULL);
}

struct xdg_surface* wmBaseGetXDGSurfaceXDG(struct xdg_wm_base* wm, struct wl_surface* sf) {
	return (struct xdg_surface*)proxyMarshalFlags(
		(struct wl_proxy*)wm, XDG_WM_BASE_GET_XDG_SURFACE, &surfaceInterfaceXDG, proxyGetVersion((struct wl_proxy*)wm), 0, NULL, sf);
}

void wmBasePongXDG(struct xdg_wm_base* wm, uint32_t serial) {
	proxyMarshalFlags((struct wl_proxy*)wm, XDG_WM_BASE_PONG, NULL, proxyGetVersion((struct wl_proxy*)wm), 0, serial);
}

void positionerDestroyXDG(struct xdg_positioner* pos) {
	proxyMarshalFlags((struct wl_proxy*)pos, XDG_POSITIONER_DESTROY, NULL, proxyGetVersion((struct wl_proxy*)pos), WL_MARSHAL_FLAG_DESTROY);
}

static void surfaceConfigure(void* data_, struct xdg_surface* xsf, uint32_t serial) {
	surfaceConfigureXDG(xsf, serial);
}

int surfaceAddListenerXDG(struct xdg_surface* xsf) {
	static const struct xdg_surface_listener ltn = { surfaceConfigure };
	return proxyAddListener((struct wl_proxy*)xsf, (void (**)(void))&ltn, NULL);
}

void surfaceDestroyXDG(struct xdg_surface* xsf) {
	proxyMarshalFlags((struct wl_proxy*)xsf, XDG_SURFACE_DESTROY, NULL, proxyGetVersion((struct wl_proxy*)xsf), WL_MARSHAL_FLAG_DESTROY);
}

struct xdg_toplevel* surfaceGetToplevelXDG(struct xdg_surface* xsf) {
	return (struct xdg_toplevel*)proxyMarshalFlags(
		(struct wl_proxy*)xsf, XDG_SURFACE_GET_TOPLEVEL, &toplevelInterfaceXDG, proxyGetVersion((struct wl_proxy*)xsf), 0, NULL);
}

struct xdg_popup* surfaceGetPopupXDG(struct xdg_surface* xsf, struct xdg_surface* parent, struct xdg_positioner* pos) {
	return (struct xdg_popup*)proxyMarshalFlags(
		(struct wl_proxy*)xsf, XDG_SURFACE_GET_POPUP, &popupInterfaceXDG, proxyGetVersion((struct wl_proxy*)xsf), 0, NULL, parent, pos);
}

void surfaceSetWindowGeometryXDG(struct xdg_surface* xsf, int32_t x, int32_t y, int32_t width, int32_t height) {
	proxyMarshalFlags((struct wl_proxy*)xsf, XDG_SURFACE_SET_WINDOW_GEOMETRY, NULL, proxyGetVersion((struct wl_proxy*)xsf), 0, x, y, width, height);
}

void surfaceAckConfigureXDG(struct xdg_surface* xsf, uint32_t serial) {
	proxyMarshalFlags((struct wl_proxy*)xsf, XDG_SURFACE_ACK_CONFIGURE, NULL, proxyGetVersion((struct wl_proxy*)xsf), 0, serial);
}

static void toplevelConfigure(void* data_, struct xdg_toplevel* tl, int32_t width, int32_t height, struct wl_array* states) {
	toplevelConfigureXDG(tl, width, height, states);
}

static void toplevelClose(void* data_, struct xdg_toplevel* tl) {
	toplevelCloseXDG(tl);
}

static void toplevelConfigureBounds(void* data_, struct xdg_toplevel* tl, int32_t width, int32_t height) {
	toplevelConfigureBoundsXDG(tl, width, height);
}

static void toplevelWMCapabilities(void* data_, struct xdg_toplevel* tl, struct wl_array* capab) {
	toplevelWMCapabilitiesXDG(tl, capab);
}

int toplevelAddListenerXDG(struct xdg_toplevel* tl) {
	static const struct xdg_toplevel_listener ltn = {
		.configure = toplevelConfigure,
		.close = toplevelClose,
		.configure_bounds = toplevelConfigureBounds,
		.wm_capabilities = toplevelWMCapabilities,
	};
	return proxyAddListener((struct wl_proxy*)tl, (void (**)(void))&ltn, NULL);
}

void toplevelDestroyXDG(struct xdg_toplevel* tl) {
	proxyMarshalFlags((struct wl_proxy*)tl, XDG_TOPLEVEL_DESTROY, NULL, proxyGetVersion((struct wl_proxy*)tl), WL_MARSHAL_FLAG_DESTROY);
}

void toplevelSetParentXDG(struct xdg_toplevel* tl, struct xdg_toplevel* parent) {
	proxyMarshalFlags((struct wl_proxy*)tl, XDG_TOPLEVEL_SET_PARENT, NULL, proxyGetVersion((struct wl_proxy*)tl), 0, parent);
}

void toplevelSetTitleXDG(struct xdg_toplevel* tl, const char* title) {
	proxyMarshalFlags((struct wl_proxy*)tl, XDG_TOPLEVEL_SET_TITLE, NULL, proxyGetVersion((struct wl_proxy*)tl), 0, title);
}

void toplevelSetAppIDXDG(struct xdg_toplevel* tl, const char* appID) {
	proxyMarshalFlags((struct wl_proxy*)tl, XDG_TOPLEVEL_SET_APP_ID, NULL, proxyGetVersion((struct wl_proxy*)tl), 0, appID);
}

void toplevelSetMaxSizeXDG(struct xdg_toplevel* tl, int32_t width, int32_t height) {
	proxyMarshalFlags((struct wl_proxy*)tl, XDG_TOPLEVEL_SET_MAX_SIZE, NULL, proxyGetVersion((struct wl_proxy*)tl), 0, width, height);
}

void toplevelSetMinSizeXDG(struct xdg_toplevel* tl, int32_t width, int32_t height) {
	proxyMarshalFlags((struct wl_proxy*)tl, XDG_TOPLEVEL_SET_MIN_SIZE, NULL, proxyGetVersion((struct wl_proxy*)tl), 0, width, height);
}

void toplevelSetFullscreenXDG(struct xdg_toplevel* tl, struct wl_output* out) {
	proxyMarshalFlags((struct wl_proxy*)tl, XDG_TOPLEVEL_SET_FULLSCREEN, NULL, proxyGetVersion((struct wl_proxy*)tl), 0, out);
}

void toplevelUnsetFullscreenXDG(struct xdg_toplevel* tl) {
	proxyMarshalFlags((struct wl_proxy*)tl, XDG_TOPLEVEL_UNSET_FULLSCREEN, NULL, proxyGetVersion((struct wl_proxy*)tl), 0);
}

static void seatCapabilities(void* data_, struct wl_seat* seat_, uint32_t capab) {
	seatCapabilitiesWayland(capab);
}

static void seatName(void* data_, struct wl_seat* seat_, const char* name) {
	char* s = strdup(name);
	if (s == NULL)
		return;
	seatNameWayland(s);
	free(s);
}

int seatAddListenerWayland(struct wl_seat *seat) {
	static const struct wl_seat_listener ltn = {
		.capabilities = seatCapabilities,
		.name = seatName,
	};
	return proxyAddListener((struct wl_proxy*)seat, (void (**)(void))&ltn, NULL);
}

void seatDestroyWayland(struct wl_seat* seat) {
	proxyDestroy((struct wl_proxy*)seat);
}

struct wl_pointer* seatGetPointerWayland(struct wl_seat* seat) {
	return (struct wl_pointer*)proxyMarshalFlags(
		(struct wl_proxy*)seat, WL_SEAT_GET_POINTER, &pointerInterfaceWayland, proxyGetVersion((struct wl_proxy*)seat), 0, NULL);
}

struct wl_keyboard* seatGetKeyboardWayland(struct wl_seat* seat) {
	return (struct wl_keyboard*)proxyMarshalFlags(
		(struct wl_proxy*)seat, WL_SEAT_GET_KEYBOARD, &keyboardInterfaceWayland, proxyGetVersion((struct wl_proxy*)seat), 0, NULL);
}

void seatReleaseWayland(struct wl_seat* seat) {
	proxyMarshalFlags((struct wl_proxy*)seat, WL_SEAT_RELEASE, NULL, proxyGetVersion((struct wl_proxy*)seat), WL_MARSHAL_FLAG_DESTROY);
}

static void pointerEnter(void* data_, struct wl_pointer* pt_, uint32_t serial, struct wl_surface* sf, wl_fixed_t x, wl_fixed_t y) {
	pointerEnterWayland(serial, sf, x, y);
}

static void pointerLeave(void* data_, struct wl_pointer* pt_, uint32_t serial, struct wl_surface* sf) {
	pointerLeaveWayland(serial, sf);
}

static void pointerMotion(void* data_, struct wl_pointer* pt_, uint32_t millis, wl_fixed_t x, wl_fixed_t y) {
	pointerMotionWayland(millis, x, y);
}

static void pointerButton(void* data_, struct wl_pointer* pt_, uint32_t serial, uint32_t millis, uint32_t button, uint32_t state) {
	pointerButtonWayland(serial, millis, button, state);
}

static void pointerAxis(void* data_, struct wl_pointer* pt_, uint32_t millis, uint32_t axis, wl_fixed_t value) {
	pointerAxisWayland(millis, axis, value);
}

static void pointerFrame(void* data_, struct wl_pointer* pt_) {
	pointerFrameWayland();
}

static void pointerAxisSource(void* data_, struct wl_pointer* pt_, uint32_t axisSrc) {
	pointerAxisSourceWayland(axisSrc);
}

static void pointerAxisStop(void* data_, struct wl_pointer* pt_, uint32_t millis, uint32_t axis) {
	pointerAxisStopWayland(millis, axis);
}

static void pointerAxisDiscrete(void* data_, struct wl_pointer* pt_, uint32_t axis, int32_t discrete) {
	pointerAxisDiscreteWayland(axis, discrete);
}

int pointerAddListenerWayland(struct wl_pointer* pt) {
	static const struct wl_pointer_listener ltn = {
		.enter = pointerEnter,
		.leave = pointerLeave,
		.motion = pointerMotion,
		.button = pointerButton,
		.axis = pointerAxis,
		.frame = pointerFrame,
		.axis_source = pointerAxisSource,
		.axis_stop = pointerAxisStop,
		.axis_discrete = pointerAxisDiscrete,
	};
	return proxyAddListener((struct wl_proxy*)pt, (void (**)(void))&ltn, NULL);
}

void pointerDestroyWayland(struct wl_pointer* pt) {
	proxyDestroy((struct wl_proxy*)pt);
}

void pointerSetCursorWayland(struct wl_pointer* pt, uint32_t serial, struct wl_surface* sf, int32_t hotspotX, int32_t hotspotY) {
	proxyMarshalFlags((struct wl_proxy*)pt, WL_POINTER_SET_CURSOR, NULL, proxyGetVersion((struct wl_proxy*)pt), 0, serial, sf, hotspotX, hotspotY);
}

void pointerReleaseWayland(struct wl_pointer* pt) {
	proxyMarshalFlags((struct wl_proxy*)pt, WL_POINTER_RELEASE, NULL, proxyGetVersion((struct wl_proxy*)pt), WL_MARSHAL_FLAG_DESTROY);
}

static void keyboardKeymap(void* data_, struct wl_keyboard* kb_, uint32_t format, int32_t fd, uint32_t size) {
	keyboardKeymapWayland(format, fd, size);
}

static void keyboardEnter(void* data_, struct wl_keyboard* kb_, uint32_t serial, struct wl_surface* sf, struct wl_array* keys) {
	keyboardEnterWayland(serial, sf, keys);
}

static void keyboardLeave(void* data_, struct wl_keyboard* kb_, uint32_t serial, struct wl_surface* sf) {
	keyboardLeaveWayland(serial, sf);
}

static void keyboardKey(void* data_, struct wl_keyboard* kb_, uint32_t serial, uint32_t millis, uint32_t key, uint32_t state) {
	keyboardKeyWayland(serial, millis, key, state);
}

static void keyboardModifiers(void* data_, struct wl_keyboard* kb_, uint32_t serial, uint32_t depressed, uint32_t latched, uint32_t locked, uint32_t group) {
	keyboardModifiersWayland(serial, depressed, latched, locked, group);
}

static void keyboardRepeatInfo(void* data_, struct wl_keyboard* kb_, int32_t rate, int32_t delay) {
	keyboardRepeatInfoWayland(rate, delay);
}

int keyboardAddListenerWayland(struct wl_keyboard* kb) {
	static const struct wl_keyboard_listener ltn = {
		.keymap = keyboardKeymap,
		.enter = keyboardEnter,
		.leave = keyboardLeave,
		.key = keyboardKey,
		.modifiers = keyboardModifiers,
		.repeat_info = keyboardRepeatInfo,
	};
	return proxyAddListener((struct wl_proxy*)kb, (void (**)(void))&ltn, NULL);
}

void keyboardDestroyWayland(struct wl_keyboard* kb) {
	proxyDestroy((struct wl_proxy*)kb);
}

void keyboardReleaseWayland(struct wl_keyboard* kb) {
	proxyMarshalFlags((struct wl_proxy*)kb, WL_KEYBOARD_RELEASE, NULL, proxyGetVersion((struct wl_proxy*)kb), WL_MARSHAL_FLAG_DESTROY);
}
