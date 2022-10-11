// Copyright 2022 Gustavo C. Viegas. All rights reserved.

//go:build linux && !android

#include <dlfcn.h>
#include <wsi_wayland.h>

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

// XXX: This is not actually version 0.
#define LIBWAYLAND "libwayland-client.so.0"

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

	return handle;

nosym:
	dlclose(handle);
	return NULL;
}

void closeWayland(void* handle) {
	dlclose(handle);
}
