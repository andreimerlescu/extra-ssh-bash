package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	sema "github.com/andreimerlescu/go-sema"
)

type PromptHistory struct {
	Commands []string
	Runtimes []time.Duration
	Outputs  [][]byte
	mu       *sync.Mutex
}

var prompt_ptr *PromptHistory

var promptMu = sync.Mutex{}

func Prompt() *PromptHistory {
	promptMu.Lock()
	defer promptMu.Unlock()
	if prompt_ptr == nil {
		prompt_ptr = &PromptHistory{
			mu:       &sync.Mutex{},
			Commands: []string{},
			Runtimes: []time.Duration{},
			Outputs:  [][]byte{},
		}
	}
	return prompt_ptr
}

func (p *PromptHistory) String() (out string) {
	var totalRuntime time.Duration
	for i := range p.Runtimes {
		totalRuntime += p.Runtimes[i]
	}

	data := make(map[int]map[string]int, 0)
	for i, command := range p.Commands {
		data[i][command] = len(p.Outputs[i])
	}

	for cmdIdx, d := range data {
		for cmd, stdOutLen := range d {
			out += "[" + fmt.Sprint(cmdIdx) + "] => " + cmd + " (STDOUT = " + fmt.Sprintf("%d", stdOutLen) + ") \n"
		}
	}
	return
}

func (p *PromptHistory) TraceIt(command string) (err error) {
	if len(command) < 2 {
		return fmt.Errorf("command must be at least 2 characters, length was %d", len(command))
	}
	if p.mu == nil {
		p.mu = &sync.Mutex{}
	}
	p.mu.Lock()
	p.Commands = append(p.Commands, command)
	p.mu.Unlock()
	return
}

func (p *PromptHistory) Run(ctx context.Context, rawCmd string, sem sema.Semaphore, env []string, handler func(CommandOutput) bool) (CommandOutput, bool) {
	if p.mu == nil {
		p.mu = &sync.Mutex{}
	}
	p.mu.Lock()
	p.Commands = append(p.Commands, rawCmd)
	p.mu.Unlock()

	innerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	commander, releaseSemaphore := Commander{}, false

	if strings.Contains(rawCmd, " | ") {

		pipes := strings.Split(rawCmd, " | ")
		var commands []UnsafeRawCommand
		for i := 0; i < len(pipes); i++ {
			pipe := pipes[i]
			commands = append(commands, UnsafeRawCommand(pipe))
		}

		if strings.Contains(rawCmd, "p4 ") {
			sem.Acquire()
			releaseSemaphore = true
		}

		co, bo := commander.Pipe(innerCtx,
			commands,
			env,
			func(stdout CommandOutput) bool {
				return handler(stdout)
			})

		if releaseSemaphore {
			sem.Release()
		}

		return co, bo

	} else {

		// pass handling this response to the func that called this func
		if strings.Contains(rawCmd, "p4 ") {
			sem.Acquire()
			releaseSemaphore = true
		}

		co, bo := commander.Run(innerCtx,
			UnsafeRawCommand(rawCmd),
			env,
			func(stdout CommandOutput) bool {
				return handler(stdout)
			})

		if releaseSemaphore {
			sem.Release()
		}

		return co, bo
	}
}

func (p *PromptHistory) RunInside(ctx context.Context, rawCmd string, sem sema.Semaphore, directory string, env []string, handler func(CommandOutput) bool) (CommandOutput, bool) {
	innerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	if strings.Contains(rawCmd, " | ") {
		return CommandOutput{Command: rawCmd, Error: errors.New("RunInside does not support commands with a pipe")}, false
	}

	err := p.TraceIt(rawCmd)
	if err != nil {
		log.Println(err)
	}

	commander, releaseSemaphore := Commander{}, false

	if strings.Contains(rawCmd, "p4 ") {
		sem.Acquire()
		releaseSemaphore = true
	}

	co, bo := commander.RunInside(innerCtx,
		UnsafeRawCommand(rawCmd),
		directory,
		env,
		func(stdout CommandOutput) bool {
			return handler(stdout)
		})

	if releaseSemaphore {
		sem.Release()
	}

	return co, bo
}

func (p *PromptHistory) RunInsideWithInput(ctx context.Context, rawCmd string, sem sema.Semaphore, directory string, input string, env []string, handler func(CommandOutput) bool) (CommandOutput, bool) {
	innerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	if strings.Contains(rawCmd, " | ") {
		return CommandOutput{Command: rawCmd, Error: errors.New("RunInside does not support commands with a pipe")}, false
	}

	err := p.TraceIt(rawCmd)
	if err != nil {
		log.Println(err)
	}

	commander, releaseSemaphore := Commander{}, false

	if strings.Contains(rawCmd, "p4 ") {
		sem.Acquire()
		releaseSemaphore = true
	}

	co, bo := commander.RunInsideWithInput(innerCtx,
		UnsafeRawCommand(rawCmd),
		directory,
		input,
		env,
		func(stdout CommandOutput) bool {
			return handler(stdout)
		})

	if releaseSemaphore {
		sem.Release()
	}

	return co, bo
}

func (p *PromptHistory) RunWithInput(ctx context.Context, rawCmd string, rawInput string, env []string, handler func(CommandOutput) bool) (CommandOutput, bool) {
	cmdr := Commander{}
	if strings.Contains(rawCmd, " | ") {
		pipes := strings.Split(rawCmd, " | ")
		var commands []UnsafeRawCommand
		for i := 0; i < len(pipes); i++ {
			commands = append(commands, UnsafeRawCommand(pipes[i]))
		}
		co, ok := cmdr.pipeWithInput(ctx,
			commands,
			bytes.NewBufferString(rawInput).Bytes(),
			env,
			func(stdout CommandOutput) bool {
				return handler(stdout)
			})
		cmdr.Stdout = co.Stdout
		return co, ok
	} else {
		co, ok := cmdr.runInsideWithInput(ctx, UnsafeRawCommand(rawCmd),
			"",
			bytes.NewBufferString(rawInput).Bytes(),
			env,
			func(stdout CommandOutput) bool {
				return handler(stdout)
			})
		cmdr.Stdout = co.Stdout
		cmdr.Stderr = co.Stderr
		return co, ok
	}
}

func (p *PromptHistory) AddCommand(rawCmd UnsafeRawCommand) {
	p.Commands = append(p.Commands, string(rawCmd))
}

func (p *PromptHistory) AddRuntime(dur time.Duration) {
	p.Runtimes = append(p.Runtimes, dur)
}

func (p *PromptHistory) AddStdout(stdout []byte) {
	p.Outputs = append(p.Outputs, stdout)
}
