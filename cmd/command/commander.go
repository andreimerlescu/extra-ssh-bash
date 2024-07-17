package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/andreimerlescu/extra-ssh-bash/cmd/data"
)

type CtxKeyPipeRawCommand string

type UnsafeRawCommand string

type Commander struct {
	Commands []Command
	Stdout   []byte
	Stderr   []byte
}

type Command struct {
	Command   string
	Cmd       *exec.Cmd
	Stdin     []byte
	Stdout    []byte
	Stderr    []byte
	ExecError error
}

type CommandOutput struct {
	Command string
	Runtime time.Duration
	Stdout  []byte
	Stderr  []byte
	Error   error
}

func (cmd *Commander) String() string {
	return fmt.Sprintf("\n %v \n OUTPUT = \n \n", string(cmd.Stdout))
}

func (cmd *Commander) Compile(command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		c := exec.Command("cmd.exe", "/c "+command)
		return c
	} else {
		fields := strings.Fields(command)
		if len(fields) < 2 {
			return exec.Command(command)
		}
		return exec.Command(fields[0], fields[1:]...)
	}
}

func (cmd *Commander) Run(ctx context.Context, rawCommand UnsafeRawCommand, env []string, handler func(CommandOutput) bool) (CommandOutput, bool) {
	return cmd.runInsideWithInput(ctx, rawCommand, "", nil, env, handler)
}

func (cmd *Commander) RunInside(ctx context.Context, rawCommand UnsafeRawCommand, directory string, env []string, handler func(CommandOutput) bool) (CommandOutput, bool) {
	return cmd.runInsideWithInput(ctx, rawCommand, directory, nil, env, handler)
}

func (cmd *Commander) RunInsideWithInput(ctx context.Context, rawCommand UnsafeRawCommand, directory string, input string, env []string, handler func(CommandOutput) bool) (CommandOutput, bool) {
	return cmd.runInsideWithInput(ctx, rawCommand, directory, bytes.NewBufferString(input).Bytes(), env, handler)
}

func (cmd *Commander) runInsideWithInput(ctx context.Context, rawCommand UnsafeRawCommand, directory string, input []byte, env []string, handler func(CommandOutput) bool) (CommandOutput, bool) {
	var _logs []string

	_logs = append(_logs, "Commander() runInsideWithInput() invoked inside...", directory)

	var (
		rawCmd  = string(rawCommand)
		output  = CommandOutput{Command: rawCmd}
		outBuff = bytes.Buffer{}
		errBuff = bytes.Buffer{}
	)

	c := cmd.Compile(rawCmd)
	if len(env) > 0 {
		c.Env = env
	}
	if len(input) > 0 {
		c.Stdin = bytes.NewReader(input)
	}

	if len(directory) > 1 {
		if !CreateDirectory(directory) {
			return CommandOutput{Command: rawCmd, Error: errors.New("directory cannot be created")}, false
		}
		_logs = append(_logs, "setting the command Dir to "+directory)
		c.Dir = directory
	} else {
		_logs = append(_logs, "setting the command Dir to <current directory> '.'")
		c.Dir = "."
	}
	c.Stdout = &outBuff
	c.Stderr = &errBuff

	_logs = append(_logs, fmt.Sprintf("PATH: %v", c.Path))
	_logs = append(_logs, fmt.Sprintf("exec command: %v", rawCmd))

	start := time.Now().UTC()
	startErr := c.Start()
	if startErr != nil {
		_logs = append(_logs, fmt.Sprint(fmt.Errorf("failed to start command with err: %v", startErr)))
		PrintLogs(_logs)
		return CommandOutput{Command: rawCmd, Stdout: outBuff.Bytes(), Stderr: errBuff.Bytes(), Error: startErr}, false
	}

	waitErr := c.Wait()
	if waitErr != nil {
		return CommandOutput{Command: rawCmd, Stdout: outBuff.Bytes(), Stderr: errBuff.Bytes(), Error: waitErr}, false
	}

	output.Stderr = errBuff.Bytes()
	output.Stdout = outBuff.Bytes()
	output.Runtime = time.Since(start)
	cmd.Stdout = output.Stdout
	cmd.Stderr = output.Stderr

	return output, handler(output)
}

func (cmd *Commander) Pipe(ctx context.Context, commands []UnsafeRawCommand, env []string, handler func(CommandOutput) bool) (CommandOutput, bool) {
	ctx = context.WithValue(ctx, CtxKeyPipeRawCommand("totalUnsafeRawCommands"), len(commands))
	return cmd.pipeWithInput(ctx, commands, nil, env, handler)
}

func (cmd *Commander) pipeWithInput(ctx context.Context, rawCmds []UnsafeRawCommand, stdin []byte, env []string, handler func(CommandOutput) bool) (CommandOutput, bool) {
	runtimeStart := time.Now().UTC()
	totalRawCmds := len(rawCmds)

	if totalRawCmds < 2 {
		log.Println("shouldn't be using the Pipe command when you are only trying to execute one "+
			"UnsafeRawCommand, the slice length was ", strconv.Itoa(totalRawCmds))
		return cmd.Run(context.WithValue(ctx, CtxKeyPipeRawCommand("error"), "no pipe in rawCmd"), rawCmds[0], env, handler)
	}

	commands := make([]Command, totalRawCmds)

	for i := 0; i < totalRawCmds; i++ {
		urc := rawCmds[i]
		commands[i].Command = string(urc)
	}

	var lastCommand Command

	for i := 0; i < totalRawCmds; i++ {
		hasPrevious := i > 0
		hasNext := i < totalRawCmds && i+1 < totalRawCmds

		previous, next := Command{}, Command{}
		if hasPrevious {
			previous = commands[i-1]
		}

		if hasNext {
			next = commands[i+1]
		}

		command := commands[i]

		command.Cmd = cmd.Compile(command.Command)
		command.Cmd.Path = os.Getenv("PATH")
		command.Cmd.Env = env
		var stdoutBuf bytes.Buffer
		var stderr bytes.Buffer
		command.Cmd.Stdout = &stdoutBuf
		command.Cmd.Stderr = &stderr

		if len(stdin) > 0 && i == 0 {
			command.Cmd.Stdin = bytes.NewReader(stdin)
		} else if len(command.Stdin) > 0 && i > 0 {
			command.Cmd.Stdin = bytes.NewReader(command.Stdin)
		}

		if hasPrevious && len(previous.Stdout) > 0 {
			command.Cmd.Stdin = bytes.NewReader(previous.Stdout)
		}

		if previous.ExecError != nil {
			fmt.Println(fmt.Errorf("previous command had an exec error, bailing with err: %v", previous.ExecError))
			return CommandOutput{}, false
		}

		Prompt().AddCommand(UnsafeRawCommand(command.Command))

		start := time.Now().UTC()

		err := command.Cmd.Run()
		if err != nil {
			log.Println(fmt.Errorf("%v: %v", "Start() error", err))
			log.Println("STDERR: ", stderr.String())
		}

		Prompt().AddRuntime(time.Since(start))

		command.Stdout = stdoutBuf.Bytes()
		command.Stderr = stderr.Bytes()
		cmd.Stdout = data.MergeByteSlices(cmd.Stdout, command.Stdout)
		cmd.Stderr = data.MergeByteSlices(cmd.Stderr, command.Stderr)

		if len(command.Stderr) > 0 {
			fmt.Println("error when running command: ", string(command.Stderr))
			return CommandOutput{}, false
		}

		if hasNext {
			next.Stdin = command.Stdout
		}
		Prompt().AddStdout(command.Stdout)

		lastCommand = command
	}

	fmt.Println("END PIPELINE TASKS\n\n ")

	cmd.Stdout = data.MergeByteSlices(cmd.Stdout, lastCommand.Stdout)
	var allCmds []string
	for i := 0; i < totalRawCmds; i++ {
		allCmds = append(allCmds, string(rawCmds[i]))
	}
	output := CommandOutput{
		Command: strings.Join(allCmds, " | "),
		Runtime: time.Since(runtimeStart),
		Stdout:  lastCommand.Stdout,
	}

	return output, handler(output)
}
