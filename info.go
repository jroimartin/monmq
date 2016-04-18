// Copyright 2015 The monmq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package monmq

import "time"

type SystemInfo struct {
	Version   string
	TotalRam  uint64
	FreeRam   uint64
	TotalSwap uint64
	FreeSwap  uint64
	CPU       float64
	Uptime    time.Duration
	Proc      ProcInfo
}

type ProcInfo struct {
	Pid      int
	TotalRam uint64
	CPU      float64
}
