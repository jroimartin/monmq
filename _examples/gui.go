// Copyright 2015 The monmq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/jroimartin/gocui"
	"github.com/jroimartin/monmq"
)

const refreshTime = 1 * time.Second

var (
	supervisor *monmq.Supervisor
	mu         sync.Mutex
)

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

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.Quit
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
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}
	return nil
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("sideTitle", -1, -1, 30, 1); err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
		fmt.Fprintf(v, "On-line agents")
	}
	if v, err := g.SetView("side", -1, 1, 30, maxY); err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
		if err := g.SetCurrentView("side"); err != nil {
			return err
		}
		v.Highlight = true
	}
	if v, err := g.SetView("mainTitle", 30, -1, maxX, 1); err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
		fmt.Fprintf(v, "General information")
	}
	if _, err := g.SetView("main", 30, 1, maxX, maxY); err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	return updateData(g)
}

func main() {
	var err error

	g := gocui.NewGui()
	if err := g.Init(); err != nil {
		log.Fatalln(err)
	}
	defer g.Close()

	g.SetLayout(layout)
	if err := keybindings(g); err != nil {
		log.Fatalln(err)
	}
	g.SelBgColor = gocui.ColorGreen
	g.SelFgColor = gocui.ColorBlack
	g.ShowCursor = true

	supervisor = monmq.NewSupervisor("amqp://amqp_broker:5672", "mon-exchange")
	if err := supervisor.Init(); err != nil {
		log.Fatalf("Init: %v", err)
	}
	defer supervisor.Shutdown()

	go func() {
		for {
			g.Flush()
			time.Sleep(refreshTime)
		}
	}()

	err = g.MainLoop()
	if err != nil && err != gocui.Quit {
		log.Fatalln(err)
	}
}

func updateData(g *gocui.Gui) error {
	mu.Lock()
	defer mu.Unlock()

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
		fmt.Fprintf(vmain, "Agent name: %v\n", agent.Name)
		fmt.Fprintf(vmain, "Running: %v\n", agent.Running)
		fmt.Fprintf(vmain, "Last heartbeat: %v\n", time.Since(agent.LastBeat))
		fmt.Fprintf(vmain, "Current tasks:\n")
		for i, t := range agent.Tasks {
			fmt.Fprintf(vmain, "%v. %v\n", i, t)
		}
	}

	return nil
}
