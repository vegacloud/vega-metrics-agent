// Copyright 2024 Vega Cloud, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

// Package utils provides utility functions for the metrics agent
package utils

import "github.com/sirupsen/logrus"

// SafeGPUMemory converts GPU memory value to bytes safely
func SafeGPUMemory(value int64) uint64 {
	const (
		GB        = uint64(1024 * 1024 * 1024)
		maxUint64 = ^uint64(0)
	)

	// First convert the value to uint64 safely
	if value < 0 {
		logrus.Warnf("Negative GPU memory value %d, using 0", value)
		return 0
	}

	uvalue := uint64(value)

	// Check if multiplication would overflow
	if uvalue > maxUint64/GB {
		logrus.Warnf("GPU memory value %d would overflow, capping at maximum", value)
		return maxUint64
	}

	return uvalue * GB
}

// SafeInt32Conversion converts int to int32 safely
func SafeInt32Conversion(value int) int32 {
	const maxInt32 = 1<<31 - 1
	const minInt32 = -1 << 31

	if value > maxInt32 {
		logrus.Warnf("Value %d exceeds maximum int32, capping at maximum", value)
		return maxInt32
	}
	if value < minInt32 {
		logrus.Warnf("Value %d is below minimum int32, capping at minimum", value)
		return minInt32
	}

	return int32(value)
}
