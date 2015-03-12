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

	softShutdown := make(chan bool)

	a = monmq.NewAgent("amqp://amqp_broker:5672", "mon-exchange", name)
	a.HardShutdownFunc = func() error {
		softShutdown <- false
		return nil
	}
	a.SoftShutdownFunc = func() error {
		softShutdown <- true
		return nil
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

	soft := <-softShutdown
	if soft {
		log.Println("Soft shutdown...")
		s.Shutdown()
		a.Shutdown()
	} else {
		log.Println("Hard shutdown...")
	}
}

func toUpper(id string, data []byte) ([]byte, error) {
	a.RegisterTask(id)
	defer a.RemoveTask(id)

	log.Printf("Received (%v): toUpper(%v)\n", id, string(data))
	time.Sleep(5 * time.Second)
	return []byte(strings.ToUpper(string(data))), nil
}
