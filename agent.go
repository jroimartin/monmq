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

// The type CommandFunction declares the signature of the methods that can be
// registered by an Agent. The data parameter contains auxiliary data.
type CommandFunction func(data []byte) ([]byte, error)

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
	SoftShutdownFunc CommandFunction

	// HardShutdownFunc will be called when a supervisor invokes the
	// command HardShutdown.
	HardShutdownFunc CommandFunction

	// PauseFunc will be called when a supervisor invokes the command
	// Pause.
	PauseFunc CommandFunction

	// ResumeFunc will be called when a supervisor invokes the command
	// Resume.
	ResumeFunc CommandFunction

	// KillTaskFunc will be called when a supervisor invokes the command
	// KillTask.
	KillTaskFunc CommandFunction
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
	if err := a.s.Register("invoke", a.invoke); err != nil {
		return err
	}
	if err := a.s.Init(); err != nil {
		return err
	}
	a.status.Running = true
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

func (a *Agent) invoke(id string, data []byte) ([]byte, error) {
	var f CommandFunction
	running := a.status.Running
	cmd, aux := Command(data[0]), data[1:]
	switch {
	case cmd == GetStatus:
		f = a.getStatus
	case a.status.Name == string(aux) && cmd == SoftShutdown:
		f = a.SoftShutdownFunc
		running = false
	case a.status.Name == string(aux) && cmd == HardShutdown:
		f = a.HardShutdownFunc
		running = false
	case a.status.Name == string(aux) && cmd == Pause:
		f = a.PauseFunc
		running = false
	case a.status.Name == string(aux) && cmd == Resume:
		f = a.ResumeFunc
		running = true
	case a.ownsTask(string(aux)) && cmd == KillTask:
		f = a.KillTaskFunc
	}
	if f == nil {
		// The command is not for this agent
		return nil, nil
	}
	b, err := f(data)
	if err != nil {
		return nil, err
	}
	a.status.Running = running
	out := fmt.Sprintf("%c%s", cmd, b)
	return []byte(out), nil
}

func (a *Agent) getStatus(data []byte) ([]byte, error) {
	return json.Marshal(a.status)
}

func (a *Agent) ownsTask(id string) bool {
	for _, t := range a.status.Tasks {
		if id == t {
			return true
		}
	}
	return false
}
