// Copyright 2023 Gustavo C. Viegas. All rights reserved.

// Package engine implements real-time rendering.
package engine

import (
	"gviegas/neo3/engine/internal/shader"
)

const (
	// The maximum number of frames in flight.
	MaxFrame = 3

	// The maximum number of lights per frame.
	MaxLight = int(shader.MaxLight)

	// The maximum number of shadow maps per frame.
	MaxShadow = int(shader.MaxShadow)

	// The maximum number of joints in a skin.
	MaxJoint = int(shader.MaxJoint)

	// The minimum size of the mesh buffer.
	MinMeshBuffer = 16384

	dflMaxDrawable       = 2048
	dflMaxMaterial       = 512
	dflMaxSkin           = 1024
	dflInitialMeshBuffer = MinMeshBuffer * 256
)

// Config is used to configure the engine.
type Config struct {
	// Prefer double-buffering rather than the
	// default triple-buffering.
	//
	// Default is false.
	DoubleBuffered bool

	// The maximum number of lights per frame.
	//
	// Default is MaxLight.
	MaxLight int

	// The maximum number of shadow maps per frame.
	//
	// Default is MaxShadow.
	MaxShadow int

	// The maximum number of joints in a skin.
	//
	// Default is MaxJoint.
	MaxJoint int

	// The maximum number of drawables per frame.
	//
	// Default is 2048.
	MaxDrawable int

	// The maximum number of materials per frame.
	//
	// Default is 512.
	MaxMaterial int

	// The maximum number of skins per frame.
	//
	// Default is 1024.
	MaxSkin int

	// The initial size of the mesh buffer.
	//
	// It must be a multiple of 16384 bytes.
	//
	// Default is 4194304 bytes (4MiB).
	InitialMeshBuffer int
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		DoubleBuffered:    false,
		MaxLight:          MaxLight,
		MaxShadow:         MaxShadow,
		MaxJoint:          MaxJoint,
		MaxDrawable:       dflMaxDrawable,
		MaxMaterial:       dflMaxMaterial,
		MaxSkin:           dflMaxSkin,
		InitialMeshBuffer: dflInitialMeshBuffer,
	}
}

var cfg Config

// Configure replaces the engine's configuration
// with config.
func Configure(config *Config) {
	// TODO...
	cfg = *config
}

func init() {
	config := DefaultConfig()
	Configure(&config)
}
