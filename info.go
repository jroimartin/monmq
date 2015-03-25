// Copyright 2015 The monmq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package monmq

import "time"

type SystemInfo struct {
	Version   string
	TotalRam  int
	FreeRam   int
	TotalSwap int
	FreeSwap  int
	CPU       float32
	Uptime    time.Duration
}
