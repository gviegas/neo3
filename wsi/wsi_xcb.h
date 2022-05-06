// Copyright 2022 Gustavo C. Viegas. All rights reserved.

// Include this file only from wsi_xcb.go.

#include <xcb/xcb.h>

// Symbol names.
const char* const nameXCB[] = {
#define CONNECT_XCB 0
	"xcb_connect",
#define DISCONNECT_XCB 1
	"xcb_disconnect",
#define FLUSH_XCB 2
	"xcb_flush",
#define CONNECTION_HAS_ERROR_XCB 3
	"xcb_connection_has_error",
#define GENERATE_ID_XCB 4
	"xcb_generate_id",
#define POLL_FOR_EVENT_XCB 5
	"xcb_poll_for_event",
#define REQUEST_CHECK_XCB 6
	"xcb_request_check",
#define GET_SETUP_XCB 7
	"xcb_get_setup",
#define SETUP_ROOTS_ITERATOR_XCB 8
	"xcb_setup_roots_iterator",
#define CREATE_WINDOW_CHECKED_XCB 9
	"xcb_create_window_checked",
#define DESTROY_WINDOW_XCB 10
	"xcb_destroy_window",
#define MAP_WINDOW_CHECKED_XCB 11
	"xcb_map_window_checked",
#define UNMAP_WINDOW_CHECKED_XCB 12
	"xcb_unmap_window_checked",
#define CONFIGURE_WINDOW_CHECKED_XCB 13
	"xcb_configure_window_checked",
#define INTERN_ATOM_XCB 14
	"xcb_intern_atom",
#define INTERN_ATOM_REPLY_XCB 15
	"xcb_intern_atom_reply",
#define CHANGE_PROPERTY_CHECKED_XCB 16
	"xcb_change_property_checked",
#define CHANGE_KEYBOARD_CONTROL_CHECKED_XCB 17
	"xcb_change_keyboard_control_checked"
};

// Symbol pointers.
void* ptrXCB[sizeof(nameXCB) / sizeof(nameXCB[0])] = {0};

// xcb_connect.
inline xcb_connection_t* connectXCB(const char* name, int* screen) {
	xcb_connection_t* (*f)(const char*, int*);
	*(void**)(&f) = ptrXCB[CONNECT_XCB];
	return f(name, screen);
}

// xcb_disconnect.
inline void disconnectXCB(xcb_connection_t* conn) {
	void (*f)(xcb_connection_t*);
	*(void**)(&f) = ptrXCB[DISCONNECT_XCB];
	f(conn);
}

// xcb_flush.
inline int flushXCB(xcb_connection_t* conn) {
	int (*f)(xcb_connection_t*);
	*(void**)(&f) = ptrXCB[FLUSH_XCB];
	return f(conn);
}

// xcb_connection_has_error.
inline int connectionHasErrorXCB(xcb_connection_t* conn) {
	int (*f)(xcb_connection_t*);
	*(void**)(&f) = ptrXCB[CONNECTION_HAS_ERROR_XCB];
	return f(conn);
}

// xcb_generate_id.
inline uint32_t generateIdXCB(xcb_connection_t* conn) {
	uint32_t (*f)(xcb_connection_t*);
	*(void**)(&f) = ptrXCB[GENERATE_ID_XCB];
	return f(conn);
}

// xcb_pool_for_event.
inline xcb_generic_event_t* pollForEventXCB(xcb_connection_t* conn) {
	xcb_generic_event_t* (*f)(xcb_connection_t*);
	*(void**)(&f) = ptrXCB[POLL_FOR_EVENT_XCB];
	return f(conn);
}

// xcb_request_check.
inline xcb_generic_error_t* requestCheckXCB(xcb_connection_t* conn, xcb_void_cookie_t cookie) {
	xcb_generic_error_t* (*f)(xcb_connection_t*, xcb_void_cookie_t);
	*(void**)(&f) = ptrXCB[REQUEST_CHECK_XCB];
	return f(conn, cookie);
}

// xcb_get_setup.
inline const struct xcb_setup_t* getSetupXCB(xcb_connection_t* conn) {
	const struct xcb_setup_t* (*f)(xcb_connection_t*);
	*(void**)(&f) = ptrXCB[GET_SETUP_XCB];
	return f(conn);
}

// xcb_setup_roots_iterartor.
inline xcb_screen_iterator_t setupRootsIteratorXCB(const xcb_setup_t* setup) {
	xcb_screen_iterator_t (*f)(const xcb_setup_t*);
	*(void**)(&f) = ptrXCB[SETUP_ROOTS_ITERATOR_XCB];
	return f(setup);
}

// xcb_create_window_checked.
inline xcb_void_cookie_t createWindowCheckedXCB(xcb_connection_t* conn, uint8_t depth, xcb_window_t id, xcb_window_t parent, int16_t x, int16_t y, uint16_t w, uint16_t h, uint16_t borderW, uint16_t class, xcb_visualid_t visual, uint32_t valMask, const void* valList) {
	xcb_void_cookie_t (*f)(xcb_connection_t*, uint8_t, xcb_window_t, xcb_window_t, int16_t, int16_t, uint16_t, uint16_t, uint16_t, uint16_t, xcb_visualid_t, uint32_t, const void*);
	*(void**)(&f) = ptrXCB[CREATE_WINDOW_CHECKED_XCB];
	return f(conn, depth, id, parent, x, y, w, h, borderW, class, visual, valMask, valList);
}

// xcb_destroy_window.
inline xcb_void_cookie_t destroyWindowXCB(xcb_connection_t* conn, xcb_window_t id) {
	xcb_void_cookie_t (*f)(xcb_connection_t*, xcb_window_t);
	*(void**)(&f) = ptrXCB[DESTROY_WINDOW_XCB];
	return f(conn, id);
}

// xcb_map_window_checked.
inline xcb_void_cookie_t mapWindowCheckedXCB(xcb_connection_t* conn, xcb_window_t id) {
	xcb_void_cookie_t (*f)(xcb_connection_t*, xcb_window_t);
	*(void**)(&f) = ptrXCB[MAP_WINDOW_CHECKED_XCB];
	return f(conn, id);
}

// xcb_unmap_window_checked.
inline xcb_void_cookie_t unmapWindowCheckedXCB(xcb_connection_t* conn, xcb_window_t id) {
	xcb_void_cookie_t (*f)(xcb_connection_t*, xcb_window_t);
	*(void**)(&f) = ptrXCB[UNMAP_WINDOW_CHECKED_XCB];
	return f(conn, id);
}

// xcb_configure_window_checked.
inline xcb_void_cookie_t configureWindowCheckedXCB(xcb_connection_t* conn, xcb_window_t id, uint32_t valMask, const void* valList) {
	xcb_void_cookie_t (*f)(xcb_connection_t*, xcb_window_t, uint32_t, const void*);
	*(void**)(&f) = ptrXCB[CONFIGURE_WINDOW_CHECKED_XCB];
	return f(conn, id, valMask, valList);
}

// xcb_intern_atom.
inline xcb_intern_atom_cookie_t internAtomXCB(xcb_connection_t* conn, uint8_t noCreate, uint16_t nameLen, const char* name) {
	xcb_intern_atom_cookie_t (*f)(xcb_connection_t*, uint8_t, uint16_t, const char*);
	*(void**)(&f) = ptrXCB[INTERN_ATOM_XCB];
	return f(conn, noCreate, nameLen, name);
}

// xcb_intern_atom_reply.
inline xcb_intern_atom_reply_t* internAtomReplyXCB(xcb_connection_t* conn, xcb_intern_atom_cookie_t cookie, xcb_generic_error_t** error) {
	xcb_intern_atom_reply_t* (*f)(xcb_connection_t*, xcb_intern_atom_cookie_t, xcb_generic_error_t**);
	*(void**)(&f) = ptrXCB[INTERN_ATOM_REPLY_XCB];
	return f(conn, cookie, error);
}

// xcb_change_property_checked.
inline xcb_void_cookie_t changePropertyCheckedXCB(xcb_connection_t* conn, uint8_t mode, xcb_window_t id, xcb_atom_t property, xcb_atom_t type, uint8_t format, uint32_t dataLen, const void* data) {
	xcb_void_cookie_t (*f)(xcb_connection_t*, uint8_t, xcb_window_t, xcb_atom_t, xcb_atom_t, uint8_t, uint32_t, const void*);
	*(void**)(&f) = ptrXCB[CHANGE_PROPERTY_CHECKED_XCB];
	return f(conn, mode, id, property, type, format, dataLen, data);
}

// xcb_change_keyboard_control_checked.
inline xcb_void_cookie_t changeKeyboardControlCheckedXCB(xcb_connection_t* conn, uint32_t valMask, const void* valList) {
	xcb_void_cookie_t (*f)(xcb_connection_t*, uint32_t, const void*);
	*(void**)(&f) = ptrXCB[CHANGE_KEYBOARD_CONTROL_CHECKED_XCB];
	return f(conn, valMask, valList);
}
