// Copyright 2022 Gustavo C. Viegas. All rights reserved.

//go:build ignore

// procgen generates code that wraps Vulkan function pointers.
package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
)

// Types for unmarshaling Vulkan procedures.
type (
	Registry struct {
		XMLName  xml.Name `xml:"registry"`
		Commands Commands `xml:"commands"`
		Enums    []Enums  `xml:"enums"` // static const vars need special care.
	}
	Commands struct {
		XMLName xml.Name  `xml:"commands"`
		Command []Command `xml:"command"`
	}
	Command struct {
		XMLName xml.Name `xml:"command"`
		Type    string   `xml:"proto>type"`
		Name    string   `xml:"proto>name"`
		Param   []Param  `xml:"param"`
		ignored bool     // Commands marked as ignored will not be loaded.
		kind    int      // Distinguishes global, instance and device commands.
		guard   string   // Conditional compilation of commands.
	}
	Param struct {
		XMLName xml.Name `xml:"param"`
		param   string   // Concatenation of <param>, <type> and <name> CharData.
	}
	Enums struct {
		XMLName xml.Name `xml:"enums"`
		Name    string   `xml:"name,attr"`
		Enum    []Enum   `xml:"enum"`
	}
	Enum struct {
		XMLName xml.Name `xml:"enum"`
		define  string   // Enum value suitable for a #define.
	}
)

// Command.kind will be set to one of these values.
const (
	Global = iota
	Instance
	Device
)

// UnmarshalXML implements xml.Unmarshaler for Param.
func (p *Param) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	ctx := "param"
tokLoop:
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "type", "name":
				if ctx != "param" {
					break tokLoop
				}
				ctx = t.Name.Local
			}

		case xml.EndElement:
			switch t.Name.Local {
			case "param":
				if ctx != "param" {
					break tokLoop
				}
				// Done.
				return nil
			case "type", "name":
				if ctx != t.Name.Local {
					break tokLoop
				}
				ctx = "param"
			}

		case xml.CharData:
			switch ctx {
			case "param":
				s := string(t)
				if strings.HasPrefix(s, "  ") {
					s = " " + strings.TrimLeft(s, " ")
				}
				if strings.HasSuffix(s, "  ") {
					s = strings.TrimRight(s, " ") + " "
				}
				p.param += s
			case "type", "name":
				p.param += string(t)
			}

		case xml.Comment:
			// Ignore.

		default:
			if err = d.Skip(); err != nil {
				return err
			}
		}
	}
	return errors.New("ill-formed XML")
}

// UnmarshalXML implements xml.Unmarshaler for Enum.
// TODO: Do not process enums that will not become macros (mostly won't).
func (e *Enum) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
attrLoop:
	for i := range start.Attr {
		if start.Attr[i].Name.Local != "name" {
			continue
		}
		n := len(start.Attr)
		for j := (i + 1) % n; j != i; j = (j + 1) % n {
			switch start.Attr[j].Name.Local {
			case "value", "alias":
				e.define = start.Attr[i].Value + " " + start.Attr[j].Value
			case "bitpos":
				e.define = start.Attr[i].Value + " (1ULL << " + start.Attr[j].Value + ")"
			default:
				continue
			}
			break attrLoop
		}
	}
	// The end token should follow.
	ctx := "enum"
tokLoop:
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.EndElement:
			switch t.Name.Local {
			case "enum":
				if ctx != "enum" {
					break tokLoop
				}
				// Done.
				return nil
			}
		case xml.CharData, xml.Comment:
			// Ignore.
		default:
			if err = d.Skip(); err != nil {
				return err
			}
		}
	}
	return errors.New("ill-formed XML")
}

// Filter marks unwanted Command elements as ignored.
func (cs *Commands) Filter() {
	want := []string{
		// From VK_KHR_surface:
		"vkDestroySurfaceKHR",
		"vkGetPhysicalDeviceSurfaceCapabilitiesKHR",
		"vkGetPhysicalDeviceSurfaceFormatsKHR",
		"vkGetPhysicalDeviceSurfacePresentModesKHR",
		"vkGetPhysicalDeviceSurfaceSupportKHR",
		// From VK_KHR_swapchain:
		"vkAcquireNextImageKHR",
		"vkCreateSwapchainKHR",
		"vkDestroySwapchainKHR",
		"vkGetSwapchainImagesKHR",
		"vkQueuePresentKHR",
	}
	wantAndroid := []string{
		// From VK_KHR_android_surface:
		"vkCreateAndroidSurfaceKHR",
	}
	wantLinux := []string{
		// From VK_KHR_wayland_surface:
		"vkCreateWaylandSurfaceKHR",
		"vkGetPhysicalDeviceWaylandPresentationSupportKHR",
		// From VK_KHR_xbc_surface:
		"vkCreateXcbSurfaceKHR",
		"vkGetPhysicalDeviceXcbPresentationSupportKHR",
	}
	wantWin32 := []string{
		// From VK_KHR_win32_surface:
		"vkCreateWin32SurfaceKHR",
		"vkGetPhysicalDeviceWin32PresentationSupportKHR",
	}
	wantGeneric := []string{
		// From VK_KHR_display:
		"vkCreateDisplayModeKHR",
		"vkCreateDisplayPlaneSurfaceKHR",
		"vkGetDisplayModePropertiesKHR",
		"vkGetDisplayPlaneCapabilitiesKHR",
		"vkGetDisplayPlaneSupportedDisplaysKHR",
		"vkGetPhysicalDeviceDisplayPlanePropertiesKHR",
		"vkGetPhysicalDeviceDisplayPropertiesKHR",
		// From VK_KHR_display_swapchain:
		"vkCreateSharedSwapchainsKHR",
	}

	// Ignore all extension commands not present in want*.
	// Names of extension commands end with an uppercase string (a tag).
	// As of v1.3, all core commands end with either a lowercase letter or a
	// number, so it is not necessary to decode the tags from XML for comparison.
cmdLoop:
	for i := range cs.Command {
		if len(cs.Command[i].Name) < 1 {
			continue
		}
		b := cs.Command[i].Name[len(cs.Command[i].Name)-1]
		if b < 65 || b > 90 {
			continue
		}
		for j := range want {
			if cs.Command[i].Name == want[j] {
				continue cmdLoop
			}
		}
		for j := range wantAndroid {
			if cs.Command[i].Name == wantAndroid[j] {
				cs.Command[i].guard = "#ifdef __ANDROID__\n"
				continue cmdLoop
			}
		}
		for j := range wantLinux {
			if cs.Command[i].Name == wantLinux[j] {
				cs.Command[i].guard = "#ifdef __linux__\n"
				continue cmdLoop
			}
		}
		for j := range wantWin32 {
			if cs.Command[i].Name == wantWin32[j] {
				cs.Command[i].guard = "#ifdef _WIN32\n"
				continue cmdLoop
			}
		}
		for j := range wantGeneric {
			if cs.Command[i].Name == wantGeneric[j] {
				cs.Command[i].guard = "#if !defined(__ANDROID__) && !defined(_WIN32)\n"
				continue cmdLoop
			}
		}
		cs.Command[i].ignored = true
	}
}

// Distinguish sets the kind of each Command element.
func (cs *Commands) Distinguish() {
	for i := range cs.Command {
		if cs.Command[i].Name == "" || len(cs.Command[i].Param) < 1 {
			continue
		}
		param := cs.Command[i].Param[0].param
		idx := strings.LastIndex(param, " ")
		if idx == -1 || idx+1 == len(param) {
			panic("bad Param format")
		}
		// Global is the default.
		switch param[:idx] {
		case "VkInstance", "VkPhysicalDevice":
			cs.Command[i].kind = Instance
		case "VkDevice", "VkQueue", "VkCommandBuffer":
			if cs.Command[i].Name != "vkGetDeviceProcAddr" {
				cs.Command[i].kind = Device
			} else {
				// vkGetDeviceProcAddr is obtained from vkGetInstanceProcAddr
				// using a valid VkInstance handle.
				cs.Command[i].kind = Instance
			}
		}
	}
}

// FPName returns the name of a function pointer variable.
func (c *Command) FPName() []byte {
	v := []byte(c.Name)[2:]
	v[0] |= 0x20
	return v
}

// GenFP generates a function pointer variable.
func (c *Command) GenFP() string {
	if c.ignored || c.Name == "" || !strings.HasPrefix(c.Name, "vk") {
		return ""
	}
	var s strings.Builder
	s.WriteString("PFN_")
	s.WriteString(c.Name)
	s.WriteByte(' ')
	s.Write(c.FPName())
	return s.String()
}

// GenFPs generates declarations/definitions of all function pointer variables.
func (cs *Commands) GenFPs(decl bool) string {
	var s strings.Builder
	for i := range cs.Command {
		v := cs.Command[i].GenFP()
		if v != "" {
			var end string
			if cs.Command[i].guard != "" {
				s.WriteString(cs.Command[i].guard)
				end = ";\n#endif\n"
			} else {
				end = ";\n"
			}
			if decl {
				s.WriteString("extern ")
				s.WriteString(v)
			} else {
				s.WriteString(v)
				s.WriteString(" = NULL")
			}
			s.WriteString(end)
		}
	}
	return s.String()[:len(s.String())-1]
}

// GenCWrapper generates a C function wrapping a function pointer call.
func (c *Command) GenCWrapper() string {
	if c.ignored || c.Name == "" || !strings.HasPrefix(c.Name, "vk") {
		return ""
	}
	var s strings.Builder
	var end string
	if c.guard != "" {
		s.WriteString(c.guard)
		end = ");\n}\n#endif"
	} else {
		end = ");\n}"
	}
	s.WriteString("static inline ")
	s.WriteString(c.Type)
	s.WriteByte(' ')
	s.WriteString(c.Name)
	s.WriteByte('(')
	for i := range c.Param {
		s.WriteString(c.Param[i].param)
		if i+1 < len(c.Param) {
			s.WriteString(", ")
		}
	}
	s.WriteString(") {\n\t")
	if c.Type != "void" {
		s.WriteString("return ")
	}
	s.Write(c.FPName())
	s.WriteByte('(')
	for i := range c.Param {
		idx := strings.LastIndex(c.Param[i].param, " ")
		if idx == -1 || idx+1 == len(c.Param[i].param) {
			panic("bad Param format")
		}
		arg := strings.Split(c.Param[i].param[idx+1:], "[")[0]
		s.WriteString(arg)
		if i+1 < len(c.Param) {
			s.WriteString(", ")
		}
	}
	s.WriteString(end)
	return s.String()
}

// GenCWrappers generates a C wrapper function for each Command in Commands.
func (cs *Commands) GenCWrappers() string {
	var s strings.Builder
	for i := range cs.Command {
		w := cs.Command[i].GenCWrapper()
		if w != "" {
			s.WriteString("\n// ")
			s.WriteString(cs.Command[i].Name)
			s.WriteByte('\n')
			s.WriteString(w)
			s.WriteByte('\n')
		}
	}
	return s.String()[:len(s.String())-1]
}

// GenCGetProc generates a C expression that obtains a function pointer.
func (c *Command) GenCGetProc() string {
	if c.ignored || c.Name == "" || !strings.HasPrefix(c.Name, "vk") {
		return ""
	}
	if c.Name == "vkGetInstanceProcAddr" {
		// vkGetInstanceProcAddr is obtained by other means.
		return ""
	}
	var s strings.Builder
	var end string
	if c.guard != "" {
		s.WriteString(c.guard)
		end = "#endif\n"
	}
	s.WriteString("\tfp = ")
	switch c.kind {
	case Global:
		s.WriteString("getInstanceProcAddr(NULL")
	case Instance:
		s.WriteString("getInstanceProcAddr(dh")
	case Device:
		s.WriteString("getDeviceProcAddr(dh")
	}
	s.WriteString(", \"")
	s.WriteString(c.Name)
	s.WriteString("\");\n\t")
	s.Write(c.FPName())
	s.WriteString(" = (PFN_")
	s.WriteString(c.Name)
	s.WriteString(")fp;\n")
	s.WriteString(end)
	return s.String()
}

// GenCGetProcs generates C functions that obtain the procedures.
func (cs *Commands) GenCGetProcs(decl bool) string {
	var s [3]strings.Builder
	s[Global].WriteString("void getGlobalProcs(void)")
	s[Instance].WriteString("void getInstanceProcs(VkInstance dh)")
	s[Device].WriteString("void getDeviceProcs(VkDevice dh)")
	if decl {
		for i := range s {
			s[i].WriteByte(';')
		}
	} else {
		for i := range s {
			s[i].WriteString(" {\n\tPFN_vkVoidFunction fp = NULL;\n")
		}
		for i := range cs.Command {
			x := cs.Command[i].GenCGetProc()
			if x == "" {
				continue
			}
			s[cs.Command[i].kind].WriteString(x)
		}
		s[Global].WriteString("}\n")
		s[Instance].WriteString("}\n")
		s[Device].WriteByte('}')
	}
	s[0].WriteByte('\n')
	s[0].WriteString(s[1].String())
	s[0].WriteByte('\n')
	s[0].WriteString(s[2].String())
	return s[0].String()
}

// GenCClearProc generates a C expression that sets a function pointer to NULL.
func (c *Command) GenCClearProc() string {
	if c.ignored || c.Name == "" || !strings.HasPrefix(c.Name, "vk") {
		return ""
	}
	if c.Name == "vkGetInstanceProcAddr" {
		// getInstanceProcAddr is left as is.
		return ""
	}
	var s strings.Builder
	var end string
	if c.guard != "" {
		s.WriteString(c.guard)
		end = "#endif\n"
	}
	s.WriteByte('\t')
	s.Write(c.FPName())
	s.WriteString(" = NULL;\n")
	s.WriteString(end)
	return s.String()
}

// GenCClearProcs geneates a C function that clears the procedures.
func (cs *Commands) GenCClearProcs(decl bool) string {
	var s strings.Builder
	s.WriteString("void clearProcs(void)")
	if decl {
		s.WriteByte(';')
	} else {
		s.WriteString(" {\n")
		for i := range cs.Command {
			x := cs.Command[i].GenCClearProc()
			if x == "" {
				continue
			}
			s.WriteString(x)
		}
		s.WriteByte('}')
	}
	return s.String()
}

// GenCMacros generates #define macros for certain enums.
func (r *Registry) GenCMacros() string {
	names := []string{"VkPipelineStageFlagBits2", "VkAccessFlagBits2"}
	var s strings.Builder
	for i := range r.Enums {
		for j := range names {
			if r.Enums[i].Name == names[j] {
				names = append(names[:j], names[j+1:]...)
				s.WriteString("\n// ")
				s.WriteString(r.Enums[i].Name)
				for j := range r.Enums[i].Enum {
					s.WriteString("\n#define ")
					s.WriteString(r.Enums[i].Enum[j].define)
				}
				s.WriteByte('\n')
				break
			}
		}
		if len(names) == 0 {
			break
		}
	}
	return s.String()
}

const CHeader = `// Code generated by procgen.go.

#ifndef PROC_H
#define PROC_H

#define VK_NO_PROTOTYPES
%s
#include <vulkan/vulkan.h>

// Function pointers.
%s

// Functions that obtain the function pointers.
// The process of obtaining the procedures for use is as follows:
//
// 1. Fetch the vkGetInstanceProcAddr symbol and assign to getInstanceProcAddr.
// 2. Call getGlobalProcs to load global procedures.
// 3. Create a valid VkInstance and use it in a call to getInstanceProcs.
// 4. Create a valid VkDevice and use it in a call to getDeviceProcs.
//
// clearProcs can be used to set all function pointers other than
// getInstanceProcAddr to NULL.
%s
%s

// Functions that wrap calls to function pointers, used by Go code.
%s

// Macros that shadow certain values defined as static constants in
// the API header, used by Go code.
%s
#endif // PROC_H
`

const CSource = `// Code generated by procgen.go.

#include <proc.h>

%s

%s

%s
`

// GenCCode generates the C header (proc.h) and the C source (proc.c).
func (r *Registry) GenCCode() (hdr, src string) {
	var s strings.Builder
	switch runtime.GOOS {
	default:
		s.WriteString("#if defined(__ANDROID__) || defined(__linux__) || defined(_WIN32)\n")
		s.WriteString("# error run procgen.go to generate the correct C files for ")
		s.WriteString(runtime.GOOS)
		s.WriteString("\n#endif\n")
		s.WriteString("// XXX: No platform-specific definitions for ")
		s.WriteString(runtime.GOOS)
	case "android":
		s.WriteString("#ifndef __ANDROID__\n")
		s.WriteString("# error run procgen.go to generate the correct C files for android\n")
		s.WriteString("#endif\n")
		s.WriteString("#define VK_USE_PLATFORM_ANDROID_KHR")
	case "linux":
		s.WriteString("#ifndef __linux__\n")
		s.WriteString("# error run procgen.go to generate the correct C files for linux\n")
		s.WriteString("#endif\n")
		s.WriteString("#define VK_USE_PLATFORM_WAYLAND_KHR\n")
		s.WriteString("#define VK_USE_PLATFORM_XCB_KHR")
	case "windows":
		s.WriteString("#ifndef _WIN32\n")
		s.WriteString("# error run procgen.go to generate the correct C files for windows\n")
		s.WriteString("#endif\n")
		s.WriteString("#define VK_USE_PLATFORM_WIN32_KHR")
	}
	cs := &r.Commands
	hdr = fmt.Sprintf(CHeader, s.String(), cs.GenFPs(true), cs.GenCGetProcs(true), cs.GenCClearProcs(true), cs.GenCWrappers(), r.GenCMacros())
	src = fmt.Sprintf(CSource, cs.GenFPs(false), cs.GenCGetProcs(false), cs.GenCClearProcs(false))
	return
}

func main() {
	if len(os.Args) <= 1 {
		os.Stderr.Write([]byte("missing argument: path/to/vk.xml\n"))
		os.Exit(1)
	}
	file, err := os.Open(os.Args[1])
	if err != nil {
		os.Stderr.Write([]byte(err.Error() + "\n"))
		os.Exit(1)
	}
	defer file.Close()

	v := &Registry{}
	err = xml.NewDecoder(file).Decode(v)
	if err != nil {
		os.Stderr.Write([]byte(err.Error() + "\n"))
		os.Exit(1)
	}
	v.Commands.Filter()
	v.Commands.Distinguish()

	chdr, csrc := v.GenCCode()
	err = os.WriteFile("proc.h", []byte(chdr), 0666)
	if err != nil {
		os.Stderr.Write([]byte(err.Error() + "\n"))
		os.Exit(1)
	}
	err = os.WriteFile("proc.c", []byte(csrc), 0666)
	if err != nil {
		os.Stderr.Write([]byte(err.Error() + "\n"))
		os.Exit(1)
	}
}
