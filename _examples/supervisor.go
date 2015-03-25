// Copyright 2015 The monmq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/jroimartin/monmq"
)

func main() {
	s := monmq.NewSupervisor("amqp://amqp_broker:5672",
		"mon-replies", "mon-exchange")
	if err := s.Init(); err != nil {
		log.Fatalf("Init: %v", err)
	}
	defer s.Shutdown()

	for {
		for _, s := range s.Status() {
			fmt.Printf("%+v\n", s)
		}
		fmt.Println("----")
		time.Sleep(1 * time.Second)
	}
}
