// Copyright 2022 Gustavo C. Viegas. All rights reserved.

// Include this file only from wsi_wayland.go.

#include <wayland-client.h>

// Symbol names.
const char* const nameWayland[] = {
#define DISPLAY_CONNECT_WAYLAND 0
	"wl_display_connect",
#define DISPLAY_DISCONNECT_WAYLAND 1
	"wl_display_disconnect",
#define DISPLAY_FLUSH_WAYLAND 2
	"wl_display_flush",
#define DISPLAY_DISPATCH_WAYLAND 3
	"wl_display_dispatch",

	// TODO
};

// Symbol pointers.
void* ptrWayland[sizeof(nameWayland) / sizeof(nameWayland[0])] = {0};

// wl_display_connect.
inline struct wl_display* displayConnectWayland(const char* name) {
	struct wl_display* (*f)(const char*);
	*(void**)(&f) = ptrWayland[DISPLAY_CONNECT_WAYLAND];
	return f(name);
}

// wl_display_disconnect.
inline void displayDisconnectWayland(struct wl_display* dpy) {
	void (*f)(struct wl_display*);
	*(void**)(&f) = ptrWayland[DISPLAY_DISCONNECT_WAYLAND];
	f(dpy);
}

// wl_display_flush.
inline int displayFlushWayland(struct wl_display* dpy) {
	int (*f)(struct wl_display*);
	*(void**)(&f) = ptrWayland[DISPLAY_FLUSH_WAYLAND];
	return f(dpy);
}

// wl_display_dispatch.
inline int displayDispatchWayland(struct wl_display* dpy) {
	int (*f)(struct wl_display*);
	*(void**)(&f) = ptrWayland[DISPLAY_DISPATCH_WAYLAND];
	return f(dpy);
}
