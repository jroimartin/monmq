// Copyright 2015 The monmq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package monmq

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jroimartin/rpcmq"
)

const sep = '|'

// Commands can be remotely invoked in workers via Supervisor.Invoke
type Command byte

const (
	GetStatus Command = iota
	SoftShutdown
	HardShutdown
	Pause
	Resume
	KillTask
)

// A Supervisor is responsible for requesting information from the deployed
// agents and sending control commands to these agents.
type Supervisor struct {
	c      *rpcmq.Client
	status []Status
	done   chan bool

	// TLSConfig allows to configure the TLS parameters used to connect to
	// the broker via amqps.
	TLSConfig *tls.Config

	// Timeout allows to configure the amount of time that must pass before
	// considering an agent as offline.
	Timeout time.Duration

	// Beat allows to establish the time between heartbeats.
	Beat time.Duration

	// Log is the logger used to register warnings and info messages. If it
	// is nil, no messages will be logged.
	Log *log.Logger
}

// Status represents the the information obtained from agents.
type Status struct {
	Name     string
	Running  bool
	Tasks    []string
	LastBeat time.Time // filled by the supervisor
}

// NewSupervisor returns a reference to a Supervisor object. The paremeter uri
// is the network address of the broker and exchange is the name of exchange
// that will be created.
func NewSupervisor(uri, exchange string) *Supervisor {
	s := &Supervisor{
		status:  []Status{},
		c:       rpcmq.NewClient(uri, "", exchange, "fanout"),
		done:    make(chan bool),
		Timeout: 5 * time.Second,
		Beat:    500 * time.Millisecond,
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
		case <-time.After(s.Beat):
			data := []byte{byte(GetStatus)}
			if _, err := s.c.Call("invoke", data, s.Timeout); err != nil {
				s.logf("GetStatus: %v", err)
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
				s.logf("route: %v", err)
			}
		case <-time.After(s.Timeout):
			s.status = []Status{}
		}
	}
}

func (s *Supervisor) route(r rpcmq.Result) error {
	if r.Err != "" {
		return errors.New(r.Err)
	}
	if len(r.Data) < 1 {
		return errors.New("malformed response")
	}

	var err error
	cmd, data := Command(r.Data[0]), r.Data[1:]
	switch cmd {
	case GetStatus:
		s.logf("GetStatus response")
		err = s.handleGetStatus(data)
	case SoftShutdown:
		s.logf("SoftShutdown response")
	case HardShutdown:
		s.logf("HardShutdown response")
	case Pause:
		s.logf("Pause response")
	case Resume:
		s.logf("Resume response")
	case KillTask:
		s.logf("KillTask response")
	default:
		err = errors.New("malformed response")
	}
	return err
}

func (s *Supervisor) handleGetStatus(data []byte) error {
	status := Status{}
	if err := json.Unmarshal(data, &status); err != nil {
		return err
	}
	status.LastBeat = time.Now()
	live := []Status{status}
	for _, st := range s.status {
		if st.Name == status.Name || time.Since(st.LastBeat) > s.Timeout {
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

// Invoke invokes the given command on the corresponding worker or task. The
// target is selected by name in the case of the workers or by uuid in the case
// of the tasks.
func (s *Supervisor) Invoke(cmd Command, target string) error {
	data := fmt.Sprintf("%c%s", cmd, target)
	if _, err := s.c.Call("invoke", []byte(data), 0); err != nil {
		return err
	}
	return nil
}

func (s *Supervisor) logf(format string, args ...interface{}) {
	if s.Log == nil {
		return
	}
	s.Log.Printf(format, args...)
}
