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

type Agent struct {
	s      *rpcmq.Server
	status Status

	TLSConfig *tls.Config

	SoftShutdown func() error
	HardShutdown func() error
	Kill         func(id string) error
}

func NewAgent(uri, exchange, name string) *Agent {
	a := &Agent{}
	a.s = rpcmq.NewServer(uri, "", exchange, "fanout")
	a.status.Name = name
	return a
}

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

func (a *Agent) Shutdown() error {
	return a.s.Shutdown()
}

func (a *Agent) RegisterTask(id string) {
	a.status.Tasks = append(a.status.Tasks, id)
}

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
	if a.SoftShutdown == nil {
		return nil, errors.New("SoftShutdown not implemented")
	}
	b := []byte(fmt.Sprintf("softShutdown%c%s", sep, name))
	err := a.SoftShutdown()
	return b, err
}

func (a *Agent) hardShutdown(id string, data []byte) ([]byte, error) {
	name := string(data)
	if name != a.status.Name {
		return nil, nil
	}
	if a.HardShutdown == nil {
		return nil, errors.New("HardShutdown not implemented")
	}
	b := []byte(fmt.Sprintf("hardShutdown%c%s", sep, name))
	err := a.HardShutdown()
	return b, err
}

func (a *Agent) kill(id string, data []byte) ([]byte, error) {
	taskID := string(data)
	if !a.ownsTask(taskID) {
		return nil, nil
	}
	if a.Kill == nil {
		return nil, errors.New("Kill not implemented")
	}
	b := []byte(fmt.Sprintf("kill%c%s", sep, taskID))
	err := a.Kill(taskID)
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
