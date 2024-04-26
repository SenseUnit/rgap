package output

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
	"time"

	"github.com/SenseUnit/rgap/config"
	"github.com/SenseUnit/rgap/iface"
	"github.com/SenseUnit/rgap/util"
)

type CommandConfig struct {
	Group   *uint64
	Command []string
	Timeout time.Duration
	NoWait  bool
}

type Command struct {
	bridge    iface.GroupBridge
	group     uint64
	command   []string
	timeout   time.Duration
	noWait    bool
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
	return &Command{
		bridge:    bridge,
		group:     *cc.Group,
		command:   cc.Command,
		timeout:   cc.Timeout,
		noWait:    cc.NoWait,
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

	var wg sync.WaitGroup

	cmd := exec.CommandContext(ctx, o.command[0], o.command[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("command %v run failed: %v", o.command, err)
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("command %v run failed: %v", o.command, err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("command %v run failed: %v", o.command, err)
		return
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer stdin.Close()
		grpItems := o.bridge.ListGroup(o.group)
		for _, item := range grpItems {
			fmt.Fprintln(stdin, item.Address().Unmap().String())
		}
	}()

	log.Printf("starting sync command %v...", o.command)
	if err := cmd.Start(); err != nil {
		log.Printf("command %v run failed: %v", o.command, err)
		return
	}

	forwardOutput := func(name string, source io.ReadCloser) {
		defer wg.Done()
		scanner := bufio.NewScanner(source)
		for scanner.Scan() {
			log.Printf("command %v %s: %s", o.command, name, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Printf("command %v %s broken: %v", o.command, name, err)
		}
	}
	wg.Add(2)
	go forwardOutput("stdout", stdout)
	go forwardOutput("stderr", stderr)
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			log.Printf("command %v exited with code %d", o.command, ee.ExitCode())
		} else {
			log.Printf("command %v run error: %v", err)
		}
	} else {
		log.Printf("command %v succeeded", o.command)
	}
}
