// Copyright 2015 The monmq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package monmq

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

const readPeriod = 100 * time.Millisecond

func readVersion() (string, error) {
	b, err := ioutil.ReadFile("/proc/version")
	if err != nil {
		return "", err
	}
	version := strings.TrimSpace(string(b))
	if version == "" {
		version = "unknown version"
	}
	return version, nil
}

type meminfo map[string]uint32

func readMeminfo() (meminfo, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info := make(meminfo)
	s := bufio.NewScanner(f)
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) < 2 {
			return nil, errors.New("malformed file")
		}
		name := fields[0][:len(fields[0])-1]
		value, err := strconv.ParseUint(fields[1], 10, 32)
		if err != nil {
			return nil, err
		}
		info[name] = uint32(value)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return info, nil
}

type cpustat struct {
	user       uint32
	nice       uint32
	system     uint32
	idle       uint32
	iowait     uint32
	irq        uint32
	softirq    uint32
	steal      uint32
	guest      uint32
	guest_nice uint32
}

func readCPUstat() (cpustat, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return cpustat{}, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if line[:3] != "cpu" {
			continue
		}
		stat := cpustat{}
		_, err := fmt.Sscanf(line[3:], "%d %d %d %d %d %d %d %d %d %d",
			&stat.user, &stat.nice, &stat.system, &stat.idle,
			&stat.iowait, &stat.irq, &stat.softirq, &stat.steal,
			&stat.guest, &stat.guest_nice)
		if err != nil {
			return cpustat{}, err
		}
		return stat, nil
	}
	if err := s.Err(); err != nil {
		return cpustat{}, err
	}
	return cpustat{}, errors.New("malformed file")
}

type procstat struct {
	pid                   int
	comm                  string
	state                 byte
	ppid                  int
	pgrp                  int
	session               int
	tty_nr                int
	tpgid                 int
	flags                 uint64
	minflt                uint32
	cminflt               uint32
	majflt                uint32
	cmajflt               uint32
	utime                 uint32
	stime                 uint32
	cutime                int32
	cstime                int32
	priority              int32
	nice                  int32
	num_threads           int32
	itrealvalue           int32
	starttime             uint64
	vsize                 uint32
	rss                   int32
	rsslim                uint32
	startcode             uint32
	endcode               uint32
	startstack            uint32
	kstkesp               uint32
	kstkeip               uint32
	signal                uint32
	blocked               uint32
	sigignore             uint32
	sigcatch              uint32
	wchan                 uint32
	nswap                 uint32
	cnswap                uint32
	exit_signal           int
	processor             int
	rt_priority           uint64
	policy                uint64
	delayacct_blkio_ticks uint64
	guest_time            uint32
	cguest_time           int32
	start_data            uint32
	end_data              uint32
	start_brk             uint32
	arg_start             uint32
	arg_end               uint32
	env_start             uint32
	env_end               uint32
	exit_code             int
}

func readProcstat(pid int) (procstat, error) {
	b, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return procstat{}, err
	}
	stat := procstat{}
	_, err = fmt.Sscanf(string(b), "%d %s %c %d %d %d %d %d %d %d %d "+
		"%d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d "+
		"%d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d "+
		"%d %d %d %d %d %d %d",
		&stat.pid, &stat.comm, &stat.state, &stat.ppid,
		&stat.pgrp, &stat.session, &stat.tty_nr, &stat.tpgid,
		&stat.flags, &stat.minflt, &stat.cminflt, &stat.majflt,
		&stat.cmajflt, &stat.utime, &stat.stime, &stat.cutime,
		&stat.cstime, &stat.priority, &stat.nice,
		&stat.num_threads, &stat.itrealvalue, &stat.starttime,
		&stat.vsize, &stat.rss, &stat.rsslim, &stat.startcode,
		&stat.endcode, &stat.startstack, &stat.kstkesp,
		&stat.kstkeip, &stat.signal, &stat.blocked,
		&stat.sigignore, &stat.sigcatch, &stat.wchan,
		&stat.nswap, &stat.cnswap, &stat.exit_signal,
		&stat.processor, &stat.rt_priority, &stat.policy,
		&stat.delayacct_blkio_ticks, &stat.guest_time,
		&stat.cguest_time, &stat.start_data, &stat.end_data,
		&stat.start_brk, &stat.arg_start, &stat.arg_end,
		&stat.env_start, &stat.env_end, &stat.exit_code)
	if err != nil {
		return procstat{}, err
	}
	return stat, nil
}

func readUptime() (int, error) {
	b, err := ioutil.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}
	var uptime int
	if _, err = fmt.Sscanf(string(b), "%d", &uptime); err != nil {
		return 0, err
	}
	return uptime, nil
}

func getSystemInfo() (SystemInfo, error) {
	si := SystemInfo{}

	version, err := readVersion()
	if err != nil {
		return SystemInfo{}, err
	}
	si.Version = version

	mi, err := readMeminfo()
	if err != nil {
		return SystemInfo{}, err
	}
	if v, ok := mi["MemTotal"]; ok {
		si.TotalRam = int(v * 1024)
	} else {
		return SystemInfo{}, errors.New("cannot get MemTotal")
	}
	if v, ok := mi["MemFree"]; ok {
		si.FreeRam = int(v * 1024)
	} else {
		return SystemInfo{}, errors.New("cannot get MemFree")
	}
	if v, ok := mi["SwapTotal"]; ok {
		si.TotalSwap = int(v * 1024)
	} else {
		return SystemInfo{}, errors.New("cannot get SwapTotal")
	}
	if v, ok := mi["SwapFree"]; ok {
		si.FreeSwap = int(v * 1024)
	} else {
		return SystemInfo{}, errors.New("cannot get SwapFree")
	}

	var totaltime, idlealltime [2]float32
	for i := 0; i < 2; i++ {
		cs, err := readCPUstat()
		if err != nil {
			return SystemInfo{}, err
		}
		// Guest time is already accounted in usertime
		usertime := float32(cs.user - cs.guest)
		nicetime := float32(cs.nice - cs.guest_nice)
		idlealltime[i] = float32(cs.idle + cs.iowait)
		systemalltime := float32(cs.system + cs.irq + cs.softirq)
		virtalltime := float32(cs.guest + cs.guest_nice)
		totaltime[i] = usertime + nicetime + systemalltime + idlealltime[i] + float32(cs.steal) + virtalltime

		if i == 0 {
			time.Sleep(readPeriod)
		}
	}
	si.CPU = ((totaltime[1] - totaltime[0]) - (idlealltime[1] - idlealltime[0])) / (totaltime[1] - totaltime[0])

	ut, err := readUptime()
	if err != nil {
		return SystemInfo{}, err
	}
	uptime, err := time.ParseDuration(strconv.Itoa(ut) + "s")
	if err != nil {
		return SystemInfo{}, err
	}
	si.Uptime = uptime

	return si, nil
}
