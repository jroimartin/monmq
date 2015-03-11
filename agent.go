// Copyright 2015 The monmq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package monmq

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jroimartin/rpcmq"
)

// An Agent is responsible for sending its status when a supervisor on the same
// exchange asks for it. It also executes the control operations requested by
// the supervisors.
type Agent struct {
	s      *rpcmq.Server
	status Status

	// TLSConfig allows to configure the TLS parameters used to connect to
	// the broker via amqps.
	TLSConfig *tls.Config

	// SoftShutdownFunc will be called when a supervisor invokes the
	// command SoftShutdown.
	SoftShutdownFunc func() error

	// HardShutdownFunc will be called when a supervisor invokes the
	// command HardShutdown.
	HardShutdownFunc func() error

	// KillFunc will be called when a supervisor invokes the command Kill.
	KillFunc func(id string) error
}

// NewAgent returns a reference to an Agent object. The paremeter uri is the
// network address of the broker and exchange is the name of exchange that will
// be created.
func NewAgent(uri, exchange, name string) *Agent {
	a := &Agent{}
	a.s = rpcmq.NewServer(uri, "", exchange, "fanout")
	a.status.Name = name
	return a
}

// Init initializes the Agent object. It establishes the connection with the
// broker, creating a channel and the exchange that will be used under the hood.
func (a *Agent) Init() error {
	a.s.TLSConfig = a.TLSConfig
	if err := a.s.Register("getStatus", a.getStatus); err != nil {
		return err
	}
	if err := a.s.Register("softShutdown", a.softShutdown); err != nil {
		return err
	}
	if err := a.s.Register("hardShutdown", a.hardShutdown); err != nil {
		return err
	}
	if err := a.s.Register("kill", a.kill); err != nil {
		return err
	}
	if err := a.s.Init(); err != nil {
		return err
	}
	return nil
}

// Shutdown shuts down the agent gracefully. Using this method will ensure that
// all requests sent by the supervisors to the agent will be received by the
// latter.
func (a *Agent) Shutdown() error {
	return a.s.Shutdown()
}

// RegisterTask adds a task to the list of tasks handled by the agent.
func (a *Agent) RegisterTask(id string) {
	a.status.Tasks = append(a.status.Tasks, id)
}

// RemoveTask removes a task from the list of tasks handled by the agent.
func (a *Agent) RemoveTask(id string) error {
	idx := -1
	for i, t := range a.status.Tasks {
		if t == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return errors.New("task not found")
	}
	a.status.Tasks = append(a.status.Tasks[:idx], a.status.Tasks[idx+1:]...)
	return nil
}

func (a *Agent) getStatus(id string, data []byte) ([]byte, error) {
	b, err := json.Marshal(a.status)
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf("getStatus%c%s", sep, b)), nil
}

func (a *Agent) softShutdown(id string, data []byte) ([]byte, error) {
	name := string(data)
	if name != a.status.Name {
		return nil, nil
	}
	if a.SoftShutdownFunc == nil {
		return nil, errors.New("SoftShutdown not implemented")
	}
	b := []byte(fmt.Sprintf("softShutdown%c%s", sep, name))
	err := a.SoftShutdownFunc()
	return b, err
}

func (a *Agent) hardShutdown(id string, data []byte) ([]byte, error) {
	name := string(data)
	if name != a.status.Name {
		return nil, nil
	}
	if a.HardShutdownFunc == nil {
		return nil, errors.New("HardShutdown not implemented")
	}
	b := []byte(fmt.Sprintf("hardShutdown%c%s", sep, name))
	err := a.HardShutdownFunc()
	return b, err
}

func (a *Agent) kill(id string, data []byte) ([]byte, error) {
	taskID := string(data)
	if !a.ownsTask(taskID) {
		return nil, nil
	}
	if a.KillFunc == nil {
		return nil, errors.New("Kill not implemented")
	}
	b := []byte(fmt.Sprintf("kill%c%s", sep, taskID))
	err := a.KillFunc(taskID)
	return b, err
}

func (a *Agent) ownsTask(id string) bool {
	for _, t := range a.status.Tasks {
		if id == t {
			return true
		}
	}
	return false
}
