// Copyright 2024 Vega Cloud, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
// File: pkg/collectors/collector.go
package collectors

import (
	"context"
)

// Collector defines the interface for all metric collectors
type Collector interface {
	// CollectMetrics collects metrics and returns them as an interface{},
	// which can be type-asserted to the specific metrics type for each collector
	CollectMetrics(ctx context.Context) (interface{}, error)
}
