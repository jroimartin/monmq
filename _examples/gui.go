// Copyright 2015 The monmq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/jroimartin/gocui"
	"github.com/jroimartin/monmq"
)

const refreshTime = 1 * time.Second

var supervisor *monmq.Supervisor

func main() {
	var err error

	g := gocui.NewGui()
	if err = g.Init(); err != nil {
		log.Fatalln(err)
	}
	defer g.Close()

	g.SetLayout(layout)
	if err = keybindings(g); err != nil {
		log.Fatalln(err)
	}
	g.SelBgColor = gocui.ColorGreen
	g.SelFgColor = gocui.ColorBlack
	g.Cursor = true

	supervisor = monmq.NewSupervisor("amqp://amqp_broker:5672",
		"mon-replies", "mon-exchange")
	if err = supervisor.Init(); err != nil {
		log.Fatalf("Init: %v", err)
	}
	defer supervisor.Shutdown()

	go func() {
		for {
			g.Execute(updateData)
			time.Sleep(refreshTime)
		}
	}()

	err = g.MainLoop()
	if err != nil && err != gocui.ErrQuit {
		log.Fatalln(err)
	}
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("side", 0, 0, 30, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		if err := g.SetCurrentView("side"); err != nil {
			return err
		}
		v.Title = "On-line agents"
		v.Highlight = true
	}
	if v, err := g.SetView("main", 30, 0, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "General information"
		v.Wrap = true
	}
	return nil
}

func keybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("side", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("side", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding("side", 'k', gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return invoke(g, v, monmq.SoftShutdown)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding("side", 'K', gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return invoke(g, v, monmq.HardShutdown)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding("side", 'p', gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return invoke(g, v, monmq.Pause)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding("side", 'r', gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return invoke(g, v, monmq.Resume)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return gocui.ErrQuit
	}); err != nil {
		return err
	}
	return nil
}

func cursorDown(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		cx, cy := v.Cursor()
		if err := v.SetCursor(cx, cy+1); err != nil {
			ox, oy := v.Origin()
			if err := v.SetOrigin(ox, oy+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func cursorUp(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		ox, oy := v.Origin()
		cx, cy := v.Cursor()
		if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
			if err := v.SetOrigin(ox, oy-1); err != nil {
				return err
			}
		}
	}
	return nil
}

func invoke(g *gocui.Gui, v *gocui.View, cmd monmq.Command) error {
	vside, err := g.View("side")
	if err != nil {
		return err
	}
	_, cy := vside.Cursor()
	selAgent, err := vside.Line(cy)
	if err != nil {
		selAgent = ""
	}
	if selAgent != "" {
		return supervisor.Invoke(cmd, selAgent)
	}
	return nil
}

func updateData(g *gocui.Gui) error {
	vside, err := g.View("side")
	if err != nil {
		return err
	}
	vmain, err := g.View("main")
	if err != nil {
		return err
	}
	vside.Clear()
	vmain.Clear()

	status := map[string]monmq.Status{}
	names := []string{}
	for _, st := range supervisor.Status() {
		status[st.Name] = st
		names = append(names, st.Name)
	}
	sort.Strings(names)

	for _, n := range names {
		fmt.Fprintf(vside, "%v\n", n)
	}

	_, cy := vside.Cursor()
	selAgent, err := vside.Line(cy)
	if err != nil {
		selAgent = ""
	}

	if agent, ok := status[selAgent]; ok {
		freeRAM := float64(agent.Info.FreeRam) / float64(agent.Info.TotalRam) * float64(100)
		freeSwap := float64(agent.Info.FreeSwap) / float64(agent.Info.TotalSwap) * float64(100)
		cpu := agent.Info.CPU * float64(100)
		procCPU := agent.Info.Proc.CPU * float64(100)
		procRam := float64(agent.Info.Proc.TotalRam) / float64(agent.Info.TotalRam) * float64(100)

		fmt.Fprintf(vmain, "Agent name: %v\n", agent.Name)
		fmt.Fprintf(vmain, "Running: %v\n", agent.Running)
		fmt.Fprintf(vmain, "Version: %s\n", agent.Info.Version)
		fmt.Fprintf(vmain, "Free RAM: %f%%, Free Swap: %f%%\n", freeRAM, freeSwap)
		fmt.Fprintf(vmain, "CPU usage: %f%%\n", cpu)
		fmt.Fprintf(vmain, "Process:\n")
		fmt.Fprintf(vmain, "  PID: %d\n", agent.Info.Proc.Pid)
		fmt.Fprintf(vmain, "  RAM usage: %f%%\n", procRam)
		fmt.Fprintf(vmain, "  CPU usage: %f%%\n", procCPU)
		fmt.Fprintf(vmain, "Uptime: %s\n", agent.Info.Uptime)
		fmt.Fprintf(vmain, "Last heartbeat: %v\n", time.Since(agent.LastBeat))
		fmt.Fprintf(vmain, "Current tasks:\n")
		for i, t := range agent.Tasks {
			fmt.Fprintf(vmain, "  %v. %v\n", i, t)
		}
	}

	return nil
}
