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
static int (*displayFlush)(struct wl_display*);
static int (*displayRoundtrip)(struct wl_display*);
static void (*proxyDestroy)(struct wl_proxy*);
static int (*proxyAddListener)(struct wl_proxy*, void (**)(void), void*);
static uint32_t (*proxyGetVersion)(struct wl_proxy*);
static struct wl_proxy* (*proxyMarshalFlags)(struct wl_proxy*, uint32_t, const struct wl_interface*, uint32_t, uint32_t, ...);

void* openWayland(void) {
	void* handle = dlopen(LIBWAYLAND, RTLD_LAZY|RTLD_GLOBAL);
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

const struct wl_interface compositorInterfaceWayland = {
	.name = "wl_compositor",
	.version = 5,
	.method_count = 2,
	.methods = (const struct wl_message[2]){
		{ "create_surface", "n", (const struct wl_interface*[1]){&surfaceInterfaceWayland} },
		{ "create_region", "n", (const struct wl_interface*[1]){&regionInterfaceWayland} },
	},
	.event_count = 0,
	.events = NULL,
};

const struct wl_interface surfaceInterfaceWayland = {
	.name = "wl_surface",
	.version = 5,
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
	.event_count = 2,
	.events = (const struct wl_message[2]){
		{ "enter", "o", (const struct wl_interface*[1]){&outputInterfaceWayland} },
		{ "leave", "o", (const struct wl_interface*[1]){&outputInterfaceWayland} },
	},
};

const struct wl_interface regionInterfaceWayland = { /* TODO */ };

const struct wl_interface outputInterfaceWayland = { /* TODO */ };

const struct wl_interface bufferInterfaceWayland = { /* TODO */ };

const struct wl_interface callbackInterfaceWayland = { /* TODO */ };

struct wl_display* displayConnectWayland(const char* name) {
	return displayConnect(name);
}

void displayDisconnectWayland(struct wl_display* dpy) {
	displayDisconnect(dpy);
}

int displayDispatchWayland(struct wl_display* dpy) {
	return displayDispatch(dpy);
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

static void registryGlobal(void*, struct wl_registry*, uint32_t name, const char* iface, uint32_t vers) {
	char* s = strdup(iface);
	if (s == NULL)
		return;
	registryGlobalWayland(name, s, vers);
	free(s);
}

static void registryGlobalRemove(void*, struct wl_registry*, uint32_t name) {
	registryGlobalRemoveWayland(name);
}

int registryAddListenerWayland(struct wl_registry* rty) {
	const struct wl_registry_listener ltn = {
		.global = registryGlobal,
		.global_remove = registryGlobalRemove,
	};
	return proxyAddListener((struct wl_proxy*)rty, (void (**)(void))&ltn, NULL);
}

void* registryBindWayland(struct wl_registry* rty, uint32_t name, const struct wl_interface* iface, uint32_t vers) {
      return proxyMarshalFlags((struct wl_proxy*)rty, WL_REGISTRY_BIND, iface, vers, 0, name, iface->name, vers, NULL);
}

struct wl_surface* compositorCreateSurfaceWayland(struct wl_compositor* cpt) {
	return (struct wl_surface*)proxyMarshalFlags(
		(struct wl_proxy*)cpt, WL_COMPOSITOR_CREATE_SURFACE, &surfaceInterfaceWayland, proxyGetVersion((struct wl_proxy*)cpt), 0, NULL);
}

static void surfaceEnter(void*, struct wl_surface* sfc, struct wl_output* out) {
	surfaceEnterWayland(sfc, out);
}

static void surfaceLeave(void*, struct wl_surface* sfc, struct wl_output* out) {
	surfaceLeaveWayland(sfc, out);
}

int surfaceAddListenerWayland(struct wl_surface* sfc) {
	const struct wl_surface_listener ltn = {
		.enter = surfaceEnter,
		.leave = surfaceLeave,
	};
	return proxyAddListener((struct wl_proxy*)sfc, (void (**)(void))&ltn, NULL);
}

void surfaceDestroyWayland(struct wl_surface* sfc) {
	proxyMarshalFlags((struct wl_proxy*)sfc, WL_SURFACE_DESTROY, NULL, proxyGetVersion((struct wl_proxy*)sfc), WL_MARSHAL_FLAG_DESTROY);
}
