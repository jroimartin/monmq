// Copyright 2015 The monmq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package monmq

import (
	"crypto/tls"
	"encoding/json"
	"errors"

	"github.com/jroimartin/rpcmq"
)

type Agent struct {
	s      *rpcmq.Server
	status Status

	TLSConfig *tls.Config
}

func NewAgent(uri, exchange, name string) *Agent {
	a := &Agent{}
	a.s = rpcmq.NewServer(uri, "", exchange, "fanout")
	a.status.Name = name
	return a
}

func (a *Agent) Init() error {
	a.s.TLSConfig = a.TLSConfig
	a.s.Register("getStatus", a.getStatus)
	if err := a.s.Init(); err != nil {
		return err
	}
	return nil
}

func (a *Agent) Shutdown() error {
	return a.s.Shutdown()
}

func (a *Agent) getStatus(id string, data []byte) ([]byte, error) {
	return json.Marshal(a.status)
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
