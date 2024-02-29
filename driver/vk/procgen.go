// Copyright 2022 Gustavo C. Viegas. All rights reserved.

//go:build ignore

// procgen generates code that wraps Vulkan function pointers.
package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Types for unmarshaling Vulkan procedures.
type (
	Registry struct {
		XMLName  xml.Name `xml:"registry"`
		Commands Commands `xml:"commands"`
		Enums    []Enums  `xml:"enums"` // static const vars need special care.
		Types    Types    `xml:"types"` // Only used for header version currently.
	}
	Commands struct {
		XMLName xml.Name  `xml:"commands"`
		Command []Command `xml:"command"`
	}
	Command struct {
		XMLName xml.Name `xml:"command"`
		API     string   `xml:"api,attr"`
		NameA   string   `xml:"name,attr"` // Only defined for aliases.
		Alias   string   `xml:"alias,attr"`
		Type    string   `xml:"proto>type"`
		Name    string   `xml:"proto>name"`
		Param   []Param  `xml:"param"`
		kind    int      // Distinguishes global, instance and device commands.
		guard   string   // Conditional compilation of commands.
	}
	Param struct {
		XMLName xml.Name `xml:"param"`
		API     string   `xml:"api,attr"`
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
	Types struct {
		XMLName xml.Name `xml:"types"`
		Type    []Type   `xml:"type"`
	}
	Type struct {
		XMLName  xml.Name `xml:"type"`
		API      string   `xml:"api,attr"`
		Name     string   `xml:"name"`
		CharData string   `xml:",chardata"`
	}
)

// Command.kind will be set to one of these values.
const (
	Global = iota
	Instance
	Device
)

// IsValidAPI checks that the API atrribute is valid ("vulkan" or undefined).
func IsValidAPI(api string) bool {
	api = strings.TrimSpace(api)
	return api == "" || api == "vulkan"
}

// UnmarshalXML implements xml.Unmarshaler for Param.
func (p *Param) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	ctx := "param"
	for i := range start.Attr {
		if start.Attr[i].Name.Local == "api" {
			p.API = start.Attr[i].Value
			if !IsValidAPI(p.API) {
				return d.Skip()
			}
		}
	}
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

// Extension commands to include.
var (
	ExtAny = []string{
		// From VK_KHR_get_physical_device_properties2:
		"vkGetPhysicalDeviceFeatures2KHR",
		"vkGetPhysicalDeviceFormatProperties2KHR",
		"vkGetPhysicalDeviceImageFormatProperties2KHR",
		"vkGetPhysicalDeviceMemoryProperties2KHR",
		"vkGetPhysicalDeviceProperties2KHR",
		"vkGetPhysicalDeviceQueueFamilyProperties2KHR",
		"vkGetPhysicalDeviceSparseImageFormatProperties2KHR",
		// From VK_KHR_create_renderpass2:
		"vkCmdBeginRenderPass2KHR",
		"vkCmdEndRenderPass2KHR",
		"vkCmdNextSubpass2KHR",
		"vkCreateRenderPass2KHR",
		// From VK_KHR_dynamic_rendering:
		"vkCmdBeginRenderingKHR",
		"vkCmdEndRenderingKHR",
		// From VK_KHR_synchronization2:
		"vkCmdPipelineBarrier2KHR",
		"vkCmdResetEvent2KHR",
		"vkCmdSetEvent2KHR",
		"vkCmdWaitEvents2KHR",
		"vkCmdWriteTimestamp2KHR",
		"vkQueueSubmit2KHR",
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
	ExtAndroid = []string{
		// From VK_KHR_android_surface:
		"vkCreateAndroidSurfaceKHR",
	}
	ExtLinux = []string{
		// From VK_KHR_wayland_surface:
		"vkCreateWaylandSurfaceKHR",
		"vkGetPhysicalDeviceWaylandPresentationSupportKHR",
	}
	ExtWin32 = []string{
		// From VK_KHR_win32_surface:
		"vkCreateWin32SurfaceKHR",
		"vkGetPhysicalDeviceWin32PresentationSupportKHR",
	}
	ExtGeneric = []string{
		// From VK_KHR_xcb_surface:
		"vkCreateXcbSurfaceKHR",
		"vkGetPhysicalDeviceXcbPresentationSupportKHR",
	}
)

// Non-extension commands to exclude.
var (
	// TODO: Decode <feature api="vulkansc"> to identify these
	// and future additions.
	VKSC = [...]string{
		"vkGetFaultData",
		"vkGetCommandPoolMemoryConsumption",
	}
)

// Filter removes unwanted commands from cs.
func (cs *Commands) Filter() {
	swapRemove := func(i int) {
		last := len(cs.Command) - 1
		cs.Command[i], cs.Command[last] = cs.Command[last], cs.Command[i]
		cs.Command = cs.Command[:last]
	}

	// Rewrite promoted extensions (i.e., aliased commands) as to use
	// their original name.
	aliases := make(map[string](int))
	for i := range cs.Command {
		if cs.Command[i].Alias != "" {
			aliases[cs.Command[i].Alias] = i
		}
	}
	for i := range cs.Command {
		if j, ok := aliases[cs.Command[i].Name]; ok {
			cs.Command[i].Name = cs.Command[j].NameA
		}
	}

	// Remove unused extension commands.
	// Names of extension commands end with an uppercase string (a tag).
	// As of v1.3, all core commands end with either a lowercase letter
	// or a number, so it is not necessary to decode the tags from XML
	// for comparison.
	var i int
cmdLoop:
	for n := len(cs.Command); n > 0; n-- {
		if len(cs.Command[i].Name) < 1 {
			// Should be an aliasing command.
			swapRemove(i)
			continue
		}
		b := cs.Command[i].Name[len(cs.Command[i].Name)-1]
		if b < 65 || b > 90 {
			for j := range VKSC {
				if cs.Command[i].Name == VKSC[j] {
					swapRemove(i)
					continue cmdLoop
				}
			}
			i++
			continue
		}
		for j := range ExtAny {
			if cs.Command[i].Name == ExtAny[j] {
				i++
				continue cmdLoop
			}
		}
		for j := range ExtAndroid {
			if cs.Command[i].Name == ExtAndroid[j] {
				cs.Command[i].guard = "#ifdef __ANDROID__\n"
				i++
				continue cmdLoop
			}
		}
		for j := range ExtLinux {
			if cs.Command[i].Name == ExtLinux[j] {
				cs.Command[i].guard = "#ifdef __linux__\n"
				i++
				continue cmdLoop
			}
		}
		for j := range ExtWin32 {
			if cs.Command[i].Name == ExtWin32[j] {
				cs.Command[i].guard = "#ifdef _WIN32\n"
				i++
				continue cmdLoop
			}
		}
		for j := range ExtGeneric {
			if cs.Command[i].Name == ExtGeneric[j] {
				cs.Command[i].guard = "#if !defined(__ANDROID__) && !defined(_WIN32)\n"
				i++
				continue cmdLoop
			}
		}
		swapRemove(i)
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
	if !strings.HasPrefix(c.Name, "vk") || !IsValidAPI(c.API) {
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
	if !strings.HasPrefix(c.Name, "vk") || !IsValidAPI(c.API) {
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
	var hasPrev bool
	for i := range c.Param {
		if !IsValidAPI(c.Param[i].API) {
			continue
		}
		if hasPrev {
			s.WriteString(", ")
		} else {
			hasPrev = true
		}
		s.WriteString(c.Param[i].param)
	}
	s.WriteString(") {\n\t")
	if c.Type != "void" {
		s.WriteString("return ")
	}
	s.Write(c.FPName())
	s.WriteByte('(')
	hasPrev = false
	for i := range c.Param {
		if !IsValidAPI(c.Param[i].API) {
			continue
		}
		idx := strings.LastIndex(c.Param[i].param, " ")
		if idx == -1 || idx+1 == len(c.Param[i].param) {
			panic("bad Param format")
		}
		arg := strings.Split(c.Param[i].param[idx+1:], "[")[0]
		if hasPrev {
			s.WriteString(", ")
		} else {
			hasPrev = true
		}
		s.WriteString(arg)
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
	if !strings.HasPrefix(c.Name, "vk") || !IsValidAPI(c.API) {
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
	if !strings.HasPrefix(c.Name, "vk") || !IsValidAPI(c.API) {
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
	names := []string{
		"VkPipelineStageFlagBits2",
		"VkAccessFlagBits2",
		"VkPipelineStageFlagBits2KHR",
		"VkAccessFlagBits2KHR",
	}
	n := len(names) / 2
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
		if len(names) == n {
			break
		}
	}
	return s.String()
}

// GenHeaderVersion generates the informational header version.
func (t *Types) GenHeaderVersion() string {
	var patch, compl string
	for i := range t.Type {
		if !IsValidAPI(t.Type[i].API) {
			continue
		}
		switch strings.TrimSpace(t.Type[i].Name) {
		case "VK_HEADER_VERSION":
			patch = strings.SplitAfter(t.Type[i].CharData, "#define")[1]
			patch = strings.TrimSpace(patch)
			if compl != "" {
				break
			}
		case "VK_HEADER_VERSION_COMPLETE":
			compl = strings.SplitAfter(t.Type[i].CharData, "#define")[1]
			compl = strings.TrimSpace(compl)
			compl = strings.Trim(compl, "()")
			if patch != "" {
				break
			}
		}
	}
	if patch == "" || compl == "" {
		panic("could not find header version in Types")
	}
	s := strings.Replace(compl, "VK_HEADER_VERSION", patch, 1)
	s = strings.ReplaceAll(s, ", ", ".")
	return s
}

const CHeader = `// Code generated by procgen.go. DO NOT EDIT.
// [vk.xml %s]

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

// Functions that wrap calls to function pointers. Used by Go code.
%s

// Macros that shadow certain values defined as static constants in
// the API header. Used by Go code.
%s
#endif // PROC_H
`

const CSource = `// Code generated by procgen.go. DO NOT EDIT.

// [vk.xml %s]

#include <proc.h>

%s

%s

%s
`

// GenCCode generates the C header (proc.h) and the C source (proc.c).
func (r *Registry) GenCCode() (hdr, src string) {
	v := r.Types.GenHeaderVersion()
	var s strings.Builder
	s.WriteString("#if !defined(__ANDROID__) && !defined(_WIN32)\n")
	s.WriteString("#define VK_USE_PLATFORM_XCB_KHR\n")
	s.WriteString("#ifdef __linux__\n")
	s.WriteString("#define VK_USE_PLATFORM_WAYLAND_KHR\n")
	s.WriteString("#endif\n")
	s.WriteString("#elif defined(__ANDROID__)\n")
	s.WriteString("#define VK_USE_PLATFORM_ANDROID_KHR\n")
	s.WriteString("#elif defined(_WIN32)\n")
	s.WriteString("#define VK_USE_PLATFORM_WIN32_KHR\n")
	s.WriteString("#endif\n")
	cs := &r.Commands
	hdr = fmt.Sprintf(CHeader, v, s.String(), cs.GenFPs(true), cs.GenCGetProcs(true), cs.GenCClearProcs(true), cs.GenCWrappers(), r.GenCMacros())
	src = fmt.Sprintf(CSource, v, cs.GenFPs(false), cs.GenCGetProcs(false), cs.GenCClearProcs(false))
	return
}

func main() {
	if len(os.Args) <= 1 {
		os.Stderr.Write([]byte("procgen.go: error: no XML input provided\n"))
		os.Exit(1)
	}
	file, err := os.Open(os.Args[1])
	if err != nil {
		os.Stderr.Write([]byte("procgen.go: error: " + err.Error() + "\n"))
		os.Exit(1)
	}
	defer file.Close()

	v := &Registry{}
	err = xml.NewDecoder(file).Decode(v)
	if err != nil {
		os.Stderr.Write([]byte("procgen.go: error: " + err.Error() + "\n"))
		os.Exit(1)
	}
	v.Commands.Filter()
	v.Commands.Distinguish()

	chdr, csrc := v.GenCCode()
	err = os.WriteFile("proc.h", []byte(chdr), 0666)
	if err != nil {
		os.Stderr.Write([]byte("procgen.go: error: " + err.Error() + "\n"))
		os.Exit(1)
	}
	err = os.WriteFile("proc.c", []byte(csrc), 0666)
	if err != nil {
		os.Stderr.Write([]byte("procgen.go: error: " + err.Error() + "\n"))
		os.Exit(1)
	}
}
