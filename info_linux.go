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

const readPeriod = 1 * time.Second

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

type meminfo map[string]uint64

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
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return nil, err
		}
		info[name] = value
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return info, nil
}

type cpustat struct {
	user       uint64
	nice       uint64
	system     uint64
	idle       uint64
	iowait     uint64
	irq        uint64
	softirq    uint64
	steal      uint64
	guest      uint64
	guest_nice uint64
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
	minflt                uint64
	cminflt               uint64
	majflt                uint64
	cmajflt               uint64
	utime                 uint64
	stime                 uint64
	cutime                int64
	cstime                int64
	priority              int64
	nice                  int64
	num_threads           int64
	itrealvalue           int64
	starttime             uint64
	vsize                 uint64
	rss                   int64
	rsslim                uint64
	startcode             uint64
	endcode               uint64
	startstack            uint64
	kstkesp               uint64
	kstkeip               uint64
	signal                uint64
	blocked               uint64
	sigignore             uint64
	sigcatch              uint64
	wchan                 uint64
	nswap                 uint64
	cnswap                uint64
	exit_signal           int
	processor             int
	rt_priority           uint64
	policy                uint64
	delayacct_blkio_ticks uint64
	guest_time            uint64
	cguest_time           int64
	start_data            uint64
	end_data              uint64
	start_brk             uint64
	arg_start             uint64
	arg_end               uint64
	env_start             uint64
	env_end               uint64
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

func readUptime() (uint64, error) {
	b, err := ioutil.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}
	var uptime uint64
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
		si.TotalRam = v * 1024
	} else {
		return SystemInfo{}, errors.New("cannot get MemTotal")
	}
	if v, ok := mi["MemFree"]; ok {
		si.FreeRam = v * 1024
	} else {
		return SystemInfo{}, errors.New("cannot get MemFree")
	}
	if v, ok := mi["SwapTotal"]; ok {
		si.TotalSwap = v * 1024
	} else {
		return SystemInfo{}, errors.New("cannot get SwapTotal")
	}
	if v, ok := mi["SwapFree"]; ok {
		si.FreeSwap = v * 1024
	} else {
		return SystemInfo{}, errors.New("cannot get SwapFree")
	}

	var (
		cs                               cpustat
		ps                               procstat
		totaltime, idlealltime, proctime [2]float64
	)
	for i := 0; i < 2; i++ {
		cs, err = readCPUstat()
		if err != nil {
			return SystemInfo{}, err
		}
		// Guest time is already accounted in usertime
		usertime := float64(cs.user - cs.guest)
		nicetime := float64(cs.nice - cs.guest_nice)
		idlealltime[i] = float64(cs.idle + cs.iowait)
		systemalltime := float64(cs.system + cs.irq + cs.softirq)
		virtalltime := float64(cs.guest + cs.guest_nice)
		totaltime[i] = usertime + nicetime + systemalltime + idlealltime[i] + float64(cs.steal) + virtalltime

		ps, err = readProcstat(os.Getpid())
		if err != nil {
			return SystemInfo{}, err
		}
		proctime[i] = float64(ps.utime + ps.stime + uint64(ps.cutime) + uint64(ps.cstime))
		if i == 0 {
			time.Sleep(readPeriod)
		}
	}
	si.CPU = ((totaltime[1] - totaltime[0]) - (idlealltime[1] - idlealltime[0])) / (totaltime[1] - totaltime[0])
	if si.CPU < 0 {
		si.CPU = 0
	}
	si.Proc.CPU = (proctime[1] - proctime[0]) / (totaltime[1] - totaltime[0])
	if si.Proc.CPU < 0 {
		si.Proc.CPU = 0
	}
	si.Proc.TotalRam = uint64(ps.rss) * uint64(os.Getpagesize())

	ut, err := readUptime()
	if err != nil {
		return SystemInfo{}, err
	}
	uptime, err := time.ParseDuration(strconv.FormatUint(ut, 10) + "s")
	if err != nil {
		return SystemInfo{}, err
	}
	si.Uptime = uptime

	si.Proc.Pid = os.Getpid()

	return si, nil
}
