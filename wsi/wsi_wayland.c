// Copyright 2022 Gustavo C. Viegas. All rights reserved.

//go:build linux && !android

#include <dlfcn.h>
#include <stdlib.h>
#include <string.h>
#include <wsi_wayland.h>
#include <_cgo_export.h>

static struct wl_interface* registryInterface;

static struct wl_display* (*displayConnect)(const char*);
struct wl_display* displayConnectWayland(const char* name) {
	return displayConnect(name);
}

static void (*displayDisconnect)(struct wl_display*);
void displayDisconnectWayland(struct wl_display* dpy) {
	displayDisconnect(dpy);
}

static int (*displayDispatch)(struct wl_display*);
int displayDispatchWayland(struct wl_display* dpy) {
	return displayDispatch(dpy);
}

static int (*displayFlush)(struct wl_display*);
int displayFlushWayland(struct wl_display* dpy) {
	return displayFlush(dpy);
}

static int (*displayRoundtrip)(struct wl_display*);
int displayRoundtripWayland(struct wl_display* dpy) {
	return displayRoundtrip(dpy);
}

static void (*proxyDestroy)(struct wl_proxy*);
static int (*proxyAddListener)(struct wl_proxy*, void (**)(void), void*);
static uint32_t (*proxyGetVersion)(struct wl_proxy*);
static struct wl_proxy* (*proxyMarshalFlags)(struct wl_proxy*, uint32_t, const struct wl_interface*, uint32_t, uint32_t, ...);

struct wl_registry* displayGetRegistryWayland(struct wl_display* dpy) {
	return (struct wl_registry*)proxyMarshalFlags(
		(struct wl_proxy*)dpy, WL_DISPLAY_GET_REGISTRY, registryInterface, proxyGetVersion((struct wl_proxy*)dpy), 0, NULL);
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
		.global        = registryGlobal,
		.global_remove = registryGlobalRemove,
	};
	return proxyAddListener((struct wl_proxy*)rty, (void (**)(void))&ltn, NULL);
}

// XXX: This is not actually version 0.
#define LIBWAYLAND "libwayland-client.so.0"

void* openWayland(void) {
	void* handle = dlopen(LIBWAYLAND, RTLD_LAZY|RTLD_GLOBAL);
	if (handle == NULL)
		return NULL;

	registryInterface = dlsym(handle, "wl_registry_interface");
	if (registryInterface == NULL)
		goto nosym;
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
