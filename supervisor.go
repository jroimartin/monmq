// Copyright 2015 The monmq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package monmq

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/jroimartin/rpcmq"
)

const sep = '|'

var (
	// Timeout allows to configure the amount of time that must pass before
	// considering an agent as offline.
	Timeout = 5 * time.Second

	// Beat allows to establish the time between heartbeats.
	Beat = 500 * time.Millisecond
)

// A Supervisor is responsible for requesting information from the deployed
// agents and sending them control commands.
type Supervisor struct {
	c      *rpcmq.Client
	status []Status
	done   chan bool

	// TLSConfig allows to configure the TLS parameters used to connect to
	// the broker via amqps
	TLSConfig *tls.Config
}

// Status represents the the information obtained from agents.
type Status struct {
	Name     string
	Tasks    []string
	LastBeat time.Time // filled by the supervisor
}

// NewSupervisor returns a reference to a Supervisor object. The paremeter uri
// is the network address of the broker and exchange is the name of exchange
// that will be created.
func NewSupervisor(uri, exchange string) *Supervisor {
	s := &Supervisor{
		status: []Status{},
		c:      rpcmq.NewClient(uri, "", exchange, "fanout"),
		done:   make(chan bool),
	}
	return s
}

// Init initializes the Supervisor object. It establishes the connection with the
// broker, creating a channel and the exchange that will be used under the hood.
func (s *Supervisor) Init() error {
	s.c.TLSConfig = s.TLSConfig
	if err := s.c.Init(); err != nil {
		return err
	}
	go s.sendHeartbeat()
	go s.getResponses()
	return nil
}

func (s *Supervisor) sendHeartbeat() {
	for {
		select {
		case <-s.done:
			return
		case <-time.After(Beat):
			if _, err := s.c.Call("getStatus", nil, Timeout); err != nil {
				log.Println("getStatus:", err)
			}
		}
	}
}

func (s *Supervisor) getResponses() {
	results := s.c.Results()
	for {
		select {
		case <-s.done:
			return
		case r := <-results:
			if err := s.route(r); err != nil {
				log.Println("route:", err)
			}
		case <-time.After(Timeout):
			s.status = []Status{}
		}
	}
}

func (s *Supervisor) route(r rpcmq.Result) error {
	if r.Err != "" {
		return errors.New(r.Err)
	}
	sdata := bytes.SplitN(r.Data, []byte{sep}, 2)
	if len(sdata) != 2 {
		return errors.New("malformed response")
	}
	cmd, data := string(sdata[0]), sdata[1]
	switch cmd {
	case "getStatus":
		return s.handleGetStatus(data)
	case "softShutdown":
		log.Println("softShutdown:", string(data))
	case "hardShutdown":
		log.Println("hardShutdown:", string(data))
	case "kill":
		log.Println("kill:", string(data))
	default:
		return errors.New("malformed response")
	}
	return nil
}

func (s *Supervisor) handleGetStatus(data []byte) error {
	status := Status{}
	if err := json.Unmarshal(data, &status); err != nil {
		return err
	}
	status.LastBeat = time.Now()
	live := []Status{status}
	for _, st := range s.status {
		if st.Name == status.Name || time.Since(st.LastBeat) > Timeout {
			continue
		}
		live = append(live, st)
	}
	s.status = live
	return nil
}

// Shutdown shuts down the supervisor gracefully. Using this method will ensure
// that all replies sent by the agents to the supervisor will be received by
// the latter.
func (s *Supervisor) Shutdown() error {
	s.done <- true // Heartbeats
	s.done <- true // Responses
	return s.c.Shutdown()
}

// Status returns the status of all the online agents.
func (s *Supervisor) Status() []Status {
	return s.status
}

// SoftShutdown invokes a soft shutdown on the agent with the specified name.
func (s *Supervisor) SoftShutdown(name string) error {
	if _, err := s.c.Call("softShutdown", []byte(name), 0); err != nil {
		log.Println("softShutdown:", err)
	}
	return nil
}

// HardShutdown invokes a hard shutdown on the agent with the specified name.
func (s *Supervisor) HardShutdown(name string) error {
	if _, err := s.c.Call("hardShutdown", []byte(name), 0); err != nil {
		log.Println("hardShutdown:", err)
	}
	return nil
}

// Kill kills the task with the id passed as parameter on the corresponding agent.
func (s *Supervisor) Kill(id string) error {
	if _, err := s.c.Call("kill", []byte(id), 0); err != nil {
		log.Println("kill:", err)
	}
	return nil
}
