// Copyright 2022 Gustavo C. Viegas. All rights reserved.

//go:build !notest

package vk

// #include <proc.h>
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

// checkCStrings checks if a string slice matches a C string array.
// It assumes that the C array, if non-nil, contains len(strs) C strings.
func checkCStrings(strs []string, cstrs unsafe.Pointer) error {
	if cstrs == nil {
		if len(strs) == 0 {
			return nil
		}
		return fmt.Errorf("checkCStrings(%v, %v): unexpected nil C array", strs, cstrs)
	}
	css := unsafe.Slice((**C.char)(cstrs), len(strs))
	for i := range strs {
		s := []byte(strs[i] + "\x00")
		for j, b := range s {
			chr := *(*C.char)(unsafe.Add(unsafe.Pointer(css[i]), j))
			if byte(chr) != b {
				const errf = "checkCStrings(%v, %v): string mismatch: \"%s\"[%d] = %c (%#v), got %c (%#v)"
				return fmt.Errorf(errf, strs, cstrs, s, j, b, b, byte(chr), byte(chr))
			}
		}
	}
	return nil
}

// checkProcOpen checks that C.getInstanceProcAddr is not nil.
func checkProcOpen() error {
	if C.getInstanceProcAddr == nil {
		return errors.New("checkInstanceProc: C.getInstanceProcAddr is nil")
	}
	return nil
}

// checkProcInstance checks the values of certain function pointers that must
// be valid after instance initialization.
func checkProcInstance() error {
	if err := checkProcOpen(); err != nil {
		return err
	}
	if C.destroyInstance == nil {
		return errors.New("checkProcInstance: C.destroyInstance is nil")
	}
	if C.createDevice == nil {
		return errors.New("checkProcInstance: C.createDevice is nil")
	}
	if C.getDeviceProcAddr == nil {
		return errors.New("checkProcInstance: C.getDeviceProcAddr is nil")
	}
	return nil
}

// checkProcDevice checks the values of certain function pointers that must
// be valid after device initialization.
func checkProcDevice() error {
	if err := checkProcInstance(); err != nil {
		return err
	}
	if C.getDeviceQueue == nil {
		return errors.New("checkProcDevice: C.getDeviceQueue is nil")
	}
	if C.queueSubmit == nil {
		return errors.New("checkProcDevice: C.queueSubmit is nil")
	}
	if C.cmdDraw == nil {
		return errors.New("checkProcDevice: C.cmdDraw is nil")
	}
	return nil
}

// checkProcClear checks the values of certain function pointers that must
// be invalid after deinitialization.
func checkProcClear() error {
	if C.getDeviceQueue != nil {
		return errors.New("checkProcClear: C.getDeviceQueue is not nil")
	}
	if C.queueSubmit != nil {
		return errors.New("checkProcClear: C.queueSubmit is not nil")
	}
	if C.cmdDraw != nil {
		return errors.New("checkProcClear: C.cmdDraw is not nil")
	}
	return nil
}
