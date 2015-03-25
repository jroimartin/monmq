// Copyright 2015 The monmq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jroimartin/monmq"
	"github.com/jroimartin/rpcmq"
)

var a *monmq.Agent

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: server name")
		os.Exit(2)
	}
	name := os.Args[1]

	cmds := make(chan monmq.Command)

	a = monmq.NewAgent("amqp://amqp_broker:5672", "mon-exchange", name)
	a.HardShutdownFunc = func(data []byte) ([]byte, error) {
		cmds <- monmq.HardShutdown
		return nil, nil
	}
	a.SoftShutdownFunc = func(data []byte) ([]byte, error) {
		cmds <- monmq.SoftShutdown
		return nil, nil
	}
	a.ResumeFunc = func(data []byte) ([]byte, error) {
		cmds <- monmq.Resume
		return nil, nil
	}
	a.PauseFunc = func(data []byte) ([]byte, error) {
		cmds <- monmq.Pause
		return nil, nil
	}
	if err := a.Init(); err != nil {
		log.Fatalf("Init: %v", err)
	}

	s := rpcmq.NewServer("amqp://amqp_broker:5672", "rcp-queue",
		"rpc-exchange", "direct")
	if err := s.Register("toUpper", toUpper); err != nil {
		log.Fatalf("Register: %v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("Init: %v", err)
	}

	paused := false
loop:
	for {
		switch <-cmds {
		case monmq.HardShutdown:
			log.Println("Hard shutdown...")
			break loop
		case monmq.SoftShutdown:
			log.Println("Soft shutdown...")
			s.Shutdown()
			a.Shutdown()
			break loop
		case monmq.Pause:
			if paused {
				continue
			}
			log.Println("Pause...")
			s.Shutdown()
			paused = true
		case monmq.Resume:
			if !paused {
				continue
			}
			log.Println("Resume...")
			if err := s.Init(); err != nil {
				log.Fatalln("Server init:", err)
			}
			paused = false
		}
	}
}

func toUpper(id string, data []byte) ([]byte, error) {
	a.RegisterTask(id)
	defer a.RemoveTask(id)

	log.Printf("Received (%v): toUpper(%v)\n", id, string(data))
	time.Sleep(5 * time.Second)
	return []byte(strings.ToUpper(string(data))), nil
}
