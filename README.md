# monmq [![GoDoc](https://godoc.org/github.com/jroimartin/monmq?status.svg)](https://godoc.org/github.com/jroimartin/monmq)

## Introduction

The Package monmq makes easier to control distributed systems based on rpcmq.

It allows to get the status of the deployed workers and their tasks, as well as
control and kill them.

## Usage

The following snippets show a minimal example on how to use monmq within a distributed system.

**Supervisor**

```go
func main() {
	...
	s := monmq.NewSupervisor("amqp://amqp_broker:5672", "mon-exchange")
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

## Installation

`go get github.com/jroimartin/monmq`

## Documentation

`godoc github.com/jroimartin/monmq`
