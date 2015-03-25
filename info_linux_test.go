// Copyright 2015 The monmq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package monmq

import (
	"os"
	"testing"
)

func TestOsVersion(t *testing.T) {
	ver, err := readVersion()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Version:", ver)
}

func TestReadMeminfo(t *testing.T) {
	mi, err := readMeminfo()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", mi)
}

func TestReadCPUStat(t *testing.T) {
	st, err := readCPUstat()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", st)
}

func TestReadProcStat(t *testing.T) {
	st, err := readProcstat(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", st)
}

func TestReadUptime(t *testing.T) {
	ut, err := readUptime()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d", ut)
}
