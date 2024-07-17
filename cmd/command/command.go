package command

import (
	"context"
	"log"
	"os/exec"
	"sync"
)

func Run(ctx context.Context, bufferSize, retries int) error {
	var (
		command = "p4 describe 4696"
		wg      sync.WaitGroup
	)

	outLw := NewLineWriter(bufferSize)
	errLw := NewLineWriter(bufferSize)

	c := exec.CommandContext(ctx, command)
	c.Dir = "."

	c.Stdout = outLw
	c.Stderr = errLw

	if err := c.Start(); err != nil {
		return err
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		for line := range outLw.Lines() {
			log.Println("Output:", line)
		}
	}()

	go func() {
		defer wg.Done()
		for line := range errLw.Lines() {
			log.Println("Error:", line)
		}
	}()

	if err := c.Wait(); err != nil {
		return err
	}

	wg.Wait()

	if err := outLw.Err(); err != nil {
		return err
	}
	if err := errLw.Err(); err != nil {
		return err
	}

	return nil
}
