# monmq [![GoDoc](https://godoc.org/github.com/jroimartin/monmq?status.svg)](https://godoc.org/github.com/jroimartin/monmq)

## Introduction

The Package monmq makes easier to control distributed systems based on rpcmq.

It allows to get the status of the deployed workers and their tasks, as well as
control and kill them.

## Getting started

The following snippets show how to use monmq within a very simple distributed
system.

**Supervisor**

```go
package main

import (
	"log"

	"github.com/jroimartin/monmq"
)

func main() {
	...
	s := monmq.NewSupervisor("amqp://amqp_broker:5672", "mon-replies", "mon-exchange")
	if err := s.Init(); err != nil {
		log.Fatalf("Init: %v", err)
	}
	defer s.Shutdown()

	go func(){
		for {
			for _, s := range s.Status() {
				// Handle status updates
			}
		}
	}()
	...
}
```

**Agent**

```go
package main

import (
	"log"

	"github.com/jroimartin/monmq"
)

var a *monmq.Agent

func main() {
	...
	a = monmq.NewAgent("amqp://amqp_broker:5672", "mon-exchange", name)
	if err := a.Init(); err != nil {
		log.Fatalf("Init: %v", err)
	}
	defer a.Shutdown()
	...
}

func rpcMethod(id string, data []byte) ([]byte, error) {
	a.RegisterTask(id)
	defer a.RemoveTask(id)

	// Method implementation
}
```

## Screenshots

![screen shot](https://cloud.githubusercontent.com/assets/1223476/6926071/3569d930-d7e4-11e4-8652-8e3ac1e0da1a.png)

## Installation

`go get github.com/jroimartin/monmq`

## Documentation

`godoc github.com/jroimartin/monmq`
