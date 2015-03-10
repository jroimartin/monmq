// Copyright 2015 The monmq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package monmq

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/jroimartin/rpcmq"
)

var (
	Timeout = 5 * time.Second
	Beat    = 500 * time.Millisecond
)

type Supervisor struct {
	c      *rpcmq.Client
	status []Status
	done   chan bool

	TLSConfig *tls.Config
}

type Status struct {
	Name     string
	Tasks    []string
	LastBeat time.Time // filled by the supervisor
}

func NewSupervisor(uri, exchange string) *Supervisor {
	s := &Supervisor{
		status: []Status{},
		c:      rpcmq.NewClient(uri, "", exchange, "fanout"),
		done:   make(chan bool),
	}
	return s
}

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
			if err := s.handleReponse(r); err != nil {
				log.Println("handleReponse:", err)
			}
		case <-time.After(Timeout):
			s.status = []Status{}
		}
	}
}

func (s *Supervisor) handleReponse(r rpcmq.Result) error {
	if r.Err != "" {
		return errors.New(r.Err)
	}
	status := Status{}
	if err := json.Unmarshal(r.Data, &status); err != nil {
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

func (s *Supervisor) Shutdown() error {
	s.done <- true // Heartbeats
	s.done <- true // Responses
	return s.c.Shutdown()
}

func (s *Supervisor) Status() []Status {
	return s.status
}

func (s *Supervisor) SoftShutdown(worker string) error {
	// TODO
	return nil
}

func (s *Supervisor) HardShutdown(worker string) error {
	// TODO
	return nil
}

func (s *Supervisor) Kill(task string) error {
	// TODO
	return nil
}
