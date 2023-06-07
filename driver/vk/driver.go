// Copyright 2022 Gustavo C. Viegas. All rights reserved.

// Package vk implements driver interfaces using the Vulkan API.
package vk

// #include <stdlib.h>
// #include <proc.h>
import "C"

import (
	"errors"
	"runtime"
	"sync"
	"unsafe"

	"gviegas/neo3/driver"
)

const driverName = "vulkan"
const preferredAPIVersion = C.VK_API_VERSION_1_3

// Driver implements driver.Driver and driver.GPU.
type Driver struct {
	proc

	inst  C.VkInstance
	ivers C.uint32_t
	pdev  C.VkPhysicalDevice
	dname string
	dvers C.uint32_t
	dev   C.VkDevice
	ques  []C.VkQueue
	qfam  C.uint32_t

	// Mutexes for ques synchronization.
	// Queue submission requires that the queue handle
	// be externally synchronized, thus this is needed
	// to allow Commit calls to run concurrently.
	qmus []sync.Mutex

	// Commit data created in advance.
	// The capacity of the channel limits the number
	// of concurrent Commit calls.
	cinfo chan *commitInfo
	csync chan *commitSync

	// Enabled extensions, indexed by ext* constants.
	exts [extN]bool

	// Used device memory, indexed by heap indices.
	mused []int64
	mprop C.VkPhysicalDeviceMemoryProperties

	// Limits of pdev.
	lim driver.Limits
}

func init() {
	driver.Register(&Driver{})
}

// initInstance initializes the Vulkan instance.
func (d *Driver) initInstance() error {
	C.getGlobalProcs()
	if C.enumerateInstanceVersion == nil || checkResult(C.vkEnumerateInstanceVersion(&d.ivers)) != nil {
		d.ivers = C.VK_API_VERSION_1_0
	}
	if isVariant(d.ivers) {
		// Do not support variants.
		return driver.ErrNoDevice
	}
	appInfo := (*C.VkApplicationInfo)(C.malloc(C.sizeof_VkApplicationInfo))
	defer C.free(unsafe.Pointer(appInfo))
	if d.ivers == C.VK_API_VERSION_1_0 {
		*appInfo = C.VkApplicationInfo{
			sType:      C.VK_STRUCTURE_TYPE_APPLICATION_INFO,
			apiVersion: C.VK_API_VERSION_1_0,
		}
	} else {
		*appInfo = C.VkApplicationInfo{
			sType:      C.VK_STRUCTURE_TYPE_APPLICATION_INFO,
			apiVersion: preferredAPIVersion,
		}
	}
	info := C.VkInstanceCreateInfo{
		sType:            C.VK_STRUCTURE_TYPE_INSTANCE_CREATE_INFO,
		pApplicationInfo: appInfo,
	}
	free, err := d.setInstanceExts(&info)
	defer free()
	if err != nil {
		return err
	}
	if err := checkResult(C.vkCreateInstance(&info, nil, &d.inst)); err != nil {
		return err
	}
	C.getInstanceProcs(d.inst)
	return nil
}

// initDevice initializes the Vulkan device.
func (d *Driver) initDevice() error {
	var n C.uint32_t
	if err := checkResult(C.vkEnumeratePhysicalDevices(d.inst, &n, nil)); err != nil {
		return err
	}
	// The wording in the spec seems to indicate that vkEnumeratePhysicalDevices
	// need not expose any devices at all. We assume that n could be zero here,
	// in which case no suitable device can be found.
	if n == 0 {
		return driver.ErrNoDevice
	}
	p := (*C.VkPhysicalDevice)(C.malloc(C.sizeof_VkPhysicalDevice * C.size_t(n)))
	defer C.free(unsafe.Pointer(p))
	if err := checkResult(C.vkEnumeratePhysicalDevices(d.inst, &n, p)); err != nil {
		return err
	}

	devs := unsafe.Slice(p, n)
	devProps := make([]C.VkPhysicalDeviceProperties, n)
	queProps := make([][]C.VkQueueFamilyProperties, n)
	for i, dev := range devs {
		C.vkGetPhysicalDeviceProperties(dev, &devProps[i])
		C.vkGetPhysicalDeviceQueueFamilyProperties(dev, &n, nil)
		p := (*C.VkQueueFamilyProperties)(C.malloc(C.sizeof_VkQueueFamilyProperties * C.size_t(n)))
		defer C.free(unsafe.Pointer(p))
		C.vkGetPhysicalDeviceQueueFamilyProperties(dev, &n, p)
		queProps[i] = unsafe.Slice(p, n)
	}

	// Select a suitable physical device to use. The bare minimum is a
	// device with a queue supporting graphics and compute operations.
	// Ideally, the device will be capable of creating swapchains and
	// be hardware-accelerated.
	weight := 0
	for i, dev := range devs {
		if isVariant(devProps[i].apiVersion) {
			// Do not support variants.
			continue
		}
		fam := len(queProps[i])
		flg := C.VkFlags(C.VK_QUEUE_GRAPHICS_BIT | C.VK_QUEUE_COMPUTE_BIT)
		for j, qp := range queProps[i] {
			if qp.queueFlags&flg == flg {
				fam = j
				break
			}
		}
		if fam == len(queProps[i]) {
			// Device does not support graphics/compute operations.
			continue
		}
		wgt := 1
		if devProps[i].deviceType&(C.VK_PHYSICAL_DEVICE_TYPE_INTEGRATED_GPU|C.VK_PHYSICAL_DEVICE_TYPE_DISCRETE_GPU) != 0 {
			wgt++
		}
		if exts, err := deviceExts(dev); err == nil {
			for _, e := range exts {
				if e == extSwapchain.name() {
					wgt += 2
					break
				}
			}
		}
		if wgt > weight {
			d.pdev = dev
			devProps[i].deviceName[len(devProps[i].deviceName)-1] = 0
			d.dname = C.GoString(&devProps[i].deviceName[0])
			d.dvers = devProps[i].apiVersion
			d.ques = make([]C.VkQueue, len(queProps[i]))
			d.qfam = C.uint32_t(fam)
			d.setLimits(&devProps[i].limits)
			weight = wgt
		}
	}
	if weight == 0 {
		// None of the exposed devices will suffice.
		return driver.ErrNoDevice
	}
	C.vkGetPhysicalDeviceMemoryProperties(d.pdev, &d.mprop)
	d.mused = make([]int64, d.mprop.memoryHeapCount)

	// Create one queue of every family exposed by the device. For graphics
	// and compute commands, the queue identified by d.qfam will be used.
	// The remaining queues only exist to increase the likelihood of finding
	// one that supports presentation.
	quePrio := (*C.float)(C.malloc(C.sizeof_float))
	defer C.free(unsafe.Pointer(quePrio))
	*quePrio = 1.0
	queInfos := (*C.VkDeviceQueueCreateInfo)(C.malloc(C.sizeof_VkDeviceQueueCreateInfo * C.size_t(len(d.ques))))
	defer C.free(unsafe.Pointer(queInfos))
	qis := unsafe.Slice(queInfos, len(d.ques))
	for i := range qis {
		qis[i] = C.VkDeviceQueueCreateInfo{
			sType:            C.VK_STRUCTURE_TYPE_DEVICE_QUEUE_CREATE_INFO,
			queueFamilyIndex: C.uint32_t(i),
			queueCount:       1,
			pQueuePriorities: quePrio,
		}
	}
	info := C.VkDeviceCreateInfo{
		sType:                C.VK_STRUCTURE_TYPE_DEVICE_CREATE_INFO,
		queueCreateInfoCount: C.uint32_t(len(d.ques)),
		pQueueCreateInfos:    queInfos,
	}
	free, err := d.setDeviceExts(&info)
	defer free()
	if err != nil {
		return err
	}
	defer d.setFeatures(&info)()
	if err := checkResult(C.vkCreateDevice(d.pdev, &info, nil, &d.dev)); err != nil {
		return err
	}
	C.getDeviceProcs(d.dev)
	for i := range d.ques {
		C.vkGetDeviceQueue(d.dev, C.uint32_t(i), 0, &d.ques[i])
	}
	return nil
}

// setLimits sets d.lim.
func (d *Driver) setLimits(lim *C.VkPhysicalDeviceLimits) {
	d.lim = driver.Limits{
		MaxImage1D:   int(lim.maxImageDimension1D),
		MaxImage2D:   int(lim.maxImageDimension2D),
		MaxImageCube: int(lim.maxImageDimensionCube),
		MaxImage3D:   int(lim.maxImageDimension3D),
		MaxLayers:    int(lim.maxImageArrayLayers),

		MaxDescHeaps:         int(lim.maxBoundDescriptorSets),
		MaxDescBuffer:        int(lim.maxPerStageDescriptorStorageBuffers),
		MaxDescImage:         int(lim.maxPerStageDescriptorStorageImages),
		MaxDescConstant:      int(lim.maxPerStageDescriptorUniformBuffers),
		MaxDescTexture:       int(lim.maxPerStageDescriptorSampledImages),
		MaxDescSampler:       int(lim.maxPerStageDescriptorSamplers),
		MaxDescBufferRange:   int64(lim.maxStorageBufferRange),
		MaxDescConstantRange: int64(lim.maxUniformBufferRange),

		MaxColorTargets: int(lim.maxColorAttachments),
		MaxRenderSize:   [2]int{int(lim.maxFramebufferWidth), int(lim.maxFramebufferHeight)},
		MaxRenderLayers: int(lim.maxFramebufferLayers),
		MaxPointSize:    float32(lim.pointSizeRange[1]),
		MaxViewports:    int(lim.maxViewports),

		MaxVertexIn:   int(lim.maxVertexInputBindings),
		MaxFragmentIn: int(lim.maxFragmentInputComponents / 4),

		MaxDispatch: [3]int{
			int(lim.maxComputeWorkGroupCount[0]),
			int(lim.maxComputeWorkGroupCount[1]),
			int(lim.maxComputeWorkGroupCount[2]),
		},
	}
}

// setFeatures chooses which features to enable.
// BUG: Either provide a way in the driver package to check what is
// enabled or just let device creation fail.
func (d *Driver) setFeatures(info *C.VkDeviceCreateInfo) (free func()) {
	var fq C.VkPhysicalDeviceFeatures
	C.vkGetPhysicalDeviceFeatures(d.pdev, &fq)
	feat := (*C.VkPhysicalDeviceFeatures)(C.malloc(C.size_t(unsafe.Sizeof(fq))))
	*feat = C.VkPhysicalDeviceFeatures{
		fullDrawIndexUint32:                     fq.fullDrawIndexUint32,
		imageCubeArray:                          fq.imageCubeArray,
		independentBlend:                        fq.independentBlend,
		depthBiasClamp:                          fq.depthBiasClamp,
		fillModeNonSolid:                        fq.fillModeNonSolid,
		largePoints:                             fq.largePoints,
		multiViewport:                           fq.multiViewport,
		samplerAnisotropy:                       fq.samplerAnisotropy,
		fragmentStoresAndAtomics:                fq.fragmentStoresAndAtomics,
		shaderUniformBufferArrayDynamicIndexing: fq.shaderUniformBufferArrayDynamicIndexing,
		shaderSampledImageArrayDynamicIndexing:  fq.shaderSampledImageArrayDynamicIndexing,
		shaderStorageBufferArrayDynamicIndexing: fq.shaderStorageBufferArrayDynamicIndexing,
		shaderStorageImageArrayDynamicIndexing:  fq.shaderStorageImageArrayDynamicIndexing,
		shaderClipDistance:                      fq.shaderClipDistance,
		shaderCullDistance:                      fq.shaderCullDistance,
	}
	info.pEnabledFeatures = feat

	// Currently, the extDynamicRendering/extSynchronization2
	// extensions are required (see ext.go).
	dynr := (*C.VkPhysicalDeviceDynamicRenderingFeaturesKHR)(C.malloc(C.sizeof_VkPhysicalDeviceDynamicRenderingFeaturesKHR))
	sync2 := (*C.VkPhysicalDeviceSynchronization2FeaturesKHR)(C.malloc(C.sizeof_VkPhysicalDeviceSynchronization2FeaturesKHR))
	*sync2 = C.VkPhysicalDeviceSynchronization2FeaturesKHR{
		sType:            C.VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_SYNCHRONIZATION_2_FEATURES_KHR,
		synchronization2: C.VK_TRUE,
	}
	*dynr = C.VkPhysicalDeviceDynamicRenderingFeaturesKHR{
		sType:            C.VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_DYNAMIC_RENDERING_FEATURES_KHR,
		pNext:            unsafe.Pointer(sync2),
		dynamicRendering: C.VK_TRUE,
	}
	proxy := (*C.VkBaseOutStructure)(unsafe.Pointer(info))
	for proxy.pNext != nil {
		proxy = proxy.pNext
	}
	proxy.pNext = (*C.VkBaseOutStructure)(unsafe.Pointer(dynr))

	return func() {
		C.free(unsafe.Pointer(feat))
		C.free(unsafe.Pointer(dynr))
		C.free(unsafe.Pointer(sync2))
	}
}

// Open initializes the driver.
func (d *Driver) Open() (gpu driver.GPU, err error) {
	if d.dev != nil {
		return d, nil
	}
	if err = d.open(); err != nil {
		goto fail
	}
	if err = d.initInstance(); err != nil {
		goto fail
	}
	if err = d.initDevice(); err != nil {
		goto fail
	}
	d.qmus = make([]sync.Mutex, len(d.ques))
	d.cinfo = make(chan *commitInfo, runtime.NumCPU())
	for i := 0; i < cap(d.cinfo); i++ {
		var ci *commitInfo
		if ci, err = d.newCommitInfo(); err != nil {
			goto fail
		}
		d.cinfo <- ci
	}
	// This channel's capacity is arbitrary.
	d.csync = make(chan *commitSync, cap(d.cinfo)*2)
	for i := 0; i < cap(d.csync); i++ {
		var cs *commitSync
		if cs, err = d.newCommitSync(); err != nil {
			goto fail
		}
		d.csync <- cs
	}
	return d, nil
fail:
	d.Close()
	return nil, err
}

// Name returns the driver name.
func (d *Driver) Name() string { return driverName }

// Close deinitializes the driver.
func (d *Driver) Close() {
	if d == nil {
		return
	}
	// We check the instance and device handles here
	// because the procs might not have been loaded.
	if d.inst != nil {
		if d.dev != nil {
			C.vkDeviceWaitIdle(d.dev)
			for len(d.cinfo) > 0 {
				d.destroyCommitInfo(<-d.cinfo)
			}
			for len(d.csync) > 0 {
				d.destroyCommitSync(<-d.csync)
			}
			// TODO: Ensure that all objects created
			// from d.dev were destroyed.
			C.vkDestroyDevice(d.dev, nil)
		}
		C.vkDestroyInstance(d.inst, nil)
	}
	C.clearProcs()
	d.close()
	*d = Driver{}
}

// memory represents a device memory allocation.
type memory struct {
	d     *Driver
	size  int64
	vis   bool
	bound bool
	p     []byte
	mem   C.VkDeviceMemory
	typ   int
	heap  int
}

// selectMemory selects a suitable memory type from the device.
// It returns the index of the selected memory, or -1 if none suffices.
func (d *Driver) selectMemory(typeBits uint, prop C.VkMemoryPropertyFlags) int {
	for i := 0; i < int(d.mprop.memoryTypeCount); i++ {
		if 1<<i&typeBits != 0 {
			flags := d.mprop.memoryTypes[i].propertyFlags
			if flags&prop == prop {
				return i
			}
		}
	}
	return -1
}

// newMemory creates a new memory allocation.
func (d *Driver) newMemory(req C.VkMemoryRequirements, visible bool) (*memory, error) {
	var prop C.VkMemoryPropertyFlags = C.VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT
	if visible {
		prop |= C.VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | C.VK_MEMORY_PROPERTY_HOST_COHERENT_BIT
	}

	typ := d.selectMemory(uint(req.memoryTypeBits), prop)
	if typ == -1 {
		// Device-local memory is desired but not required.
		prop &^= C.VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT
	}
	typ = d.selectMemory(uint(req.memoryTypeBits), prop)
	if typ == -1 {
		return nil, errors.New("vk: no suitable memory type found")
	}

	info := C.VkMemoryAllocateInfo{
		sType:           C.VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
		allocationSize:  req.size,
		memoryTypeIndex: C.uint32_t(typ),
	}
	var mem C.VkDeviceMemory
	if err := checkResult(C.vkAllocateMemory(d.dev, &info, nil, &mem)); err != nil {
		return nil, err
	}
	heap := int(d.mprop.memoryTypes[typ].heapIndex)
	d.mused[heap] += int64(req.size)

	return &memory{
		d:    d,
		size: int64(req.size),
		vis:  visible,
		mem:  mem,
		typ:  typ,
		heap: heap,
	}, nil
}

// mmap maps the memory for host access.
// The memory must be host visible (m.vis) and must have been bound to a
// resource (m.bound).
func (m *memory) mmap() error {
	if !m.vis {
		panic("cannot map memory that is not host visible")
	}
	if !m.bound {
		panic("cannot map memory that is not bound to a resource")
	}
	if len(m.p) == 0 {
		var p unsafe.Pointer
		if err := checkResult(C.vkMapMemory(m.d.dev, m.mem, 0, C.VK_WHOLE_SIZE, 0, &p)); err != nil {
			return err
		}
		m.p = unsafe.Slice((*byte)(p), m.size)
	}
	return nil
}

// unmap unmaps the memory.
func (m *memory) unmap() {
	if len(m.p) != 0 {
		C.vkUnmapMemory(m.d.dev, m.mem)
		m.p = nil
	}
}

// free deallocates and invalidates the memory.
func (m *memory) free() {
	if m == nil {
		return
	}
	if m.d != nil {
		C.vkFreeMemory(m.d.dev, m.mem, nil)
		m.d.mused[m.heap] -= m.size
	}
	*m = memory{}
}

// Driver returns the receiver (for driver.GPU conformance).
func (d *Driver) Driver() driver.Driver { return d }

// Limits returns the implementation limits.
func (d *Driver) Limits() driver.Limits { return d.lim }

// checkResult returns an error derived from a VkResult value.
// If such value does not indicate an error, it returns nil instead.
func checkResult(res C.VkResult) error {
	if res >= 0 {
		// Not an error: VK_ERROR_* values are all negative.
		return nil
	}
	switch res {
	case C.VK_ERROR_OUT_OF_HOST_MEMORY:
		return errNoHostMemory
	case C.VK_ERROR_OUT_OF_DEVICE_MEMORY:
		return errNoDeviceMemory
	case C.VK_ERROR_INITIALIZATION_FAILED:
		return errInitFailed
	case C.VK_ERROR_DEVICE_LOST:
		return errDeviceLost
	case C.VK_ERROR_MEMORY_MAP_FAILED:
		return errMMapFailed
	case C.VK_ERROR_LAYER_NOT_PRESENT:
		return errNoLayer
	case C.VK_ERROR_EXTENSION_NOT_PRESENT:
		return errNoExtension
	case C.VK_ERROR_FEATURE_NOT_PRESENT:
		return errNoFeature
	case C.VK_ERROR_INCOMPATIBLE_DRIVER:
		return errDriverCompat
	case C.VK_ERROR_TOO_MANY_OBJECTS:
		return errTooManyObjects
	case C.VK_ERROR_FORMAT_NOT_SUPPORTED:
		return errUnsupportedFormat
	case C.VK_ERROR_FRAGMENTED_POOL:
		return errFragmentedPool
	case C.VK_ERROR_OUT_OF_POOL_MEMORY:
		return errNoPoolMemory
	case C.VK_ERROR_INVALID_EXTERNAL_HANDLE:
		return errExternalHandle
	case C.VK_ERROR_FRAGMENTATION:
		return errFragmentation
	case C.VK_ERROR_SURFACE_LOST_KHR:
		return errSurfaceLost
	case C.VK_ERROR_NATIVE_WINDOW_IN_USE_KHR:
		return errWindowInUse
	case C.VK_ERROR_OUT_OF_DATE_KHR:
		return errOutOfDate
	case C.VK_ERROR_INCOMPATIBLE_DISPLAY_KHR:
		return errDisplayCompat
	}
	return errUnknown
}

// Common Vulkan errors (VK_ERROR_*).
var (
	errNoHostMemory      = driver.ErrNoHostMemory
	errNoDeviceMemory    = driver.ErrNoDeviceMemory
	errInitFailed        = errors.New("vk: initialization failed")
	errDeviceLost        = driver.ErrFatal
	errMMapFailed        = errors.New("vk: memory map failed")
	errNoLayer           = errors.New("vk: layer not present")
	errNoExtension       = errors.New("vk: extension not present")
	errNoFeature         = errors.New("vk: feature not present")
	errDriverCompat      = errors.New("vk: incompatible driver")
	errTooManyObjects    = errors.New("vk: too many objects")
	errUnsupportedFormat = errors.New("vk: format not supported")
	errFragmentedPool    = errors.New("vk: fragmented pool")
	errUnknown           = errors.New("vk: unknown error")
	errNoPoolMemory      = errors.New("vk: out of pool memory")
	errExternalHandle    = errors.New("vk: invalid external handle")
	errFragmentation     = errors.New("vk: fragmentation")
	errSurfaceLost       = errors.New("vk: surface lost")
	errWindowInUse       = errors.New("vk: native window in use")
	errOutOfDate         = driver.ErrSwapchain
	errDisplayCompat     = errors.New("vk: incompatible display")
)

// DeviceName returns the name of the VkDevice that the driver
// is using.
func (d *Driver) DeviceName() string { return d.dname }

// InstanceVersion returns the version of the VkInstance that
// the driver is using.
func (d *Driver) InstanceVersion() (major, minor, patch int) {
	major = versionMajor(d.ivers)
	minor = versionMinor(d.ivers)
	patch = versionPatch(d.ivers)
	return
}

// DeviceVersion returns the version of the VkDevice that
// the driver is using.
func (d *Driver) DeviceVersion() (major, minor, patch int) {
	major = versionMajor(d.dvers)
	minor = versionMinor(d.dvers)
	patch = versionPatch(d.dvers)
	return
}

// versionMajor extracts the major version number from v.
// v must have been generated by VK_MAKE_API_VERSION.
func versionMajor(v C.uint32_t) int { return int(v >> 22 & 0x7f) }

// versionMinor extracts the minor version number from v.
// v must have been generated by VK_MAKE_API_VERSION.
func versionMinor(v C.uint32_t) int { return int(v >> 12 & 0x3ff) }

// versionPatch extracts the patch version number from v.
// v must have been generated by VK_MAKE_API_VERSION.
func versionPatch(v C.uint32_t) int { return int(v & 0xfff) }

// isVariant returns whether version v identifies a variant
// implementation of the Vulkan API.
// v must have been generated by VK_MAKE_API_VERSION.
func isVariant(v C.uint32_t) bool { return v>>29 != 0 }
