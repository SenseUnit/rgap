package output

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"

	"github.com/SenseUnit/rgap/config"
	"github.com/SenseUnit/rgap/iface"
	"github.com/SenseUnit/rgap/util"
)

type CommandConfig struct {
	Group     *uint64
	Command   []string
	Timeout   time.Duration
	NoWait    bool
	WaitDelay *time.Duration `yaml:"wait_delay"`
}

type Command struct {
	bridge    iface.GroupBridge
	group     uint64
	command   []string
	timeout   time.Duration
	noWait    bool
	waitDelay time.Duration
	syncQueue chan struct{}
	shutdown  chan struct{}
	busy      sync.WaitGroup
	unsubFns  []func()
}

func NewCommand(cfg *config.OutputConfig, bridge iface.GroupBridge) (*Command, error) {
	var cc CommandConfig
	if err := util.CheckedUnmarshal(&cfg.Spec, &cc); err != nil {
		return nil, fmt.Errorf("cannot unmarshal command output config: %w", err)
	}
	if cc.Group == nil {
		return nil, errors.New("group is not specified")
	}
	if len(cc.Command) == 0 {
		return nil, errors.New("command is not specified")
	}
	waitDelay := 100 * time.Millisecond
	if cc.WaitDelay != nil {
		waitDelay = *cc.WaitDelay
	}
	return &Command{
		bridge:    bridge,
		group:     *cc.Group,
		command:   cc.Command,
		timeout:   cc.Timeout,
		noWait:    cc.NoWait,
		waitDelay: waitDelay,
		syncQueue: make(chan struct{}, 1),
		shutdown:  make(chan struct{}),
	}, nil
}

func (o *Command) Start() error {
	if !o.noWait {
		// This addition to WaitGroup must happen before any Wait()
		// therefore it is synchronized with startup and holds back
		// delivery of events before
		o.busy.Add(1)
		go func() {
			defer o.busy.Done()
			o.syncLoop()
		}()
	}
	o.unsubFns = append(o.unsubFns,
		o.bridge.OnJoin(o.group, func(group uint64, item iface.GroupItem) {
			o.sync()
		}),
		o.bridge.OnLeave(o.group, func(group uint64, item iface.GroupItem) {
			o.sync()
		}),
	)
	log.Println("started command output plugin")
	return nil
}

func (o *Command) Stop() error {
	for _, unsub := range o.unsubFns {
		unsub()
	}
	close(o.shutdown)
	log.Println("command output plugin stopping - waiting commands to finish..")
	o.busy.Wait()
	log.Println("stopped command output plugin")
	return nil
}

func (o *Command) sync() {
	if o.noWait {
		// This addition to WaitGroup must happen any Wait()
		// therefore it is synchronized with event subscription
		// funcs and holds them back. This way we can be sure that
		// if unsub funcs invoked before Wait(), all additions are
		// synchronized before Wait().
		o.busy.Add(1)
		go func() {
			defer o.busy.Done()
			o.runCommand()
		}()
	} else {
		select {
		case o.syncQueue <- struct{}{}:
		default:
		}
	}
}

func (o *Command) syncLoop() {
	for {
		select {
		case <-o.shutdown:
			return
		case <-o.syncQueue:
			o.runCommand()
		}
	}
}

func (o *Command) runCommand() {
	ctx := context.Background()
	if o.timeout > 0 {
		ctx1, cancel := context.WithTimeout(ctx, o.timeout)
		defer cancel()
		ctx = ctx1
	}

	cmd := exec.CommandContext(ctx, o.command[0], o.command[1:]...)
	cmd.WaitDelay = o.waitDelay

	var stdinBuf bytes.Buffer
	for _, item := range o.bridge.ListGroup(o.group) {
		fmt.Fprintln(&stdinBuf, item.Address().Unmap().String())
	}
	cmd.Stdin = &stdinBuf

	err := func() error {
		stdout := newOutputForwarder("stdout", o.command)
		defer stdout.Close()
		cmd.Stdout = stdout

		stderr := newOutputForwarder("stderr", o.command)
		defer stderr.Close()
		cmd.Stderr = stderr

		log.Printf("starting sync command %v...", o.command)
		return cmd.Run()
	}()

	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			log.Printf("command %v exited with code %d", o.command, ee.ExitCode())
		} else {
			log.Printf("command %v run error: %v", o.command, err)
		}
	} else {
		log.Printf("command %v succeeded", o.command)
	}
}

type outputForwarder struct {
	name    string
	command []string
	buf     []byte
}

func newOutputForwarder(name string, command []string) *outputForwarder {
	return &outputForwarder{
		name:    name,
		command: command,
	}
}

func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

func (of *outputForwarder) Write(p []byte) (int, error) {
	n := len(p)
	for i := bytes.IndexByte(p, '\n'); i >= 0; i = bytes.IndexByte(p, '\n') {
		yield := dropCR(p[:i])
		if len(of.buf) > 0 {
			log.Printf("command %v %s: %s%s", of.command, of.name, of.buf, yield)
			of.buf = nil
		} else {
			log.Printf("command %v %s: %s", of.command, of.name, yield)
		}
		p = p[i+1:]
	}
	if len(p) > 0 {
		of.buf = make([]byte, len(p))
		copy(of.buf, p)
	}
	return n, nil
}

func (of *outputForwarder) Close() error {
	if len(of.buf) > 0 {
		log.Printf("command %v %s: %s", of.command, of.name, of.buf)
		of.buf = nil
	}
	return nil
}
