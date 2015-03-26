// Copyright 2015 The monmq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package monmq

import "testing"

func TestSystemInfo(t *testing.T) {
	info, err := getSystemInfo()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Version: %s", info.Version)
	t.Logf("TotalRam: %d, FreeRam: %d, TotalSwap: %d, FreeSwap: %d",
		info.TotalRam, info.FreeRam, info.TotalSwap, info.FreeSwap)
	t.Logf("CPU: %f, Uptime: %s",
		info.CPU, info.Uptime)
	t.Logf("Proc.Pid: %d", info.Proc.Pid)
	t.Logf("Proc.TotalRam: %d", info.Proc.TotalRam)
	t.Logf("Proc.CPU: %f", info.Proc.CPU)
}
