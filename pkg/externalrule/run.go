package externalrule

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

type Program struct {
	Command     string
	Arguments   []string
	Directory   string
	Environment []string
	Timeout     time.Duration
}

func Run(ctx context.Context, program Program, request Request) (Response, error) {
	if ctx == nil {
		return Response{}, fmt.Errorf("external rule: nil context")
	}
	if program.Command == "" {
		return Response{}, fmt.Errorf("external rule: empty command")
	}
	if request.ProtocolVersion == 0 {
		request.ProtocolVersion = ProtocolVersion
	}
	if request.ProtocolVersion != ProtocolVersion {
		return Response{}, fmt.Errorf("external rule: unsupported request protocol version %d", request.ProtocolVersion)
	}
	input, err := json.Marshal(request)
	if err != nil {
		return Response{}, fmt.Errorf("external rule: encode request: %w", err)
	}
	if program.Timeout <= 0 {
		program.Timeout = 10 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, program.Timeout)
	defer cancel()
	command := exec.CommandContext(runCtx, program.Command, program.Arguments...)
	command.Dir = program.Directory
	if len(program.Environment) != 0 {
		command.Env = append(os.Environ(), program.Environment...)
	}
	command.Stdin = bytes.NewReader(input)
	var stdout, stderr limitedBuffer
	stdout.limit = 8 << 20
	stderr.limit = 1 << 20
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		if runCtx.Err() != nil {
			return Response{}, fmt.Errorf("external rule: %w", runCtx.Err())
		}
		if stderr.Len() != 0 {
			return Response{}, fmt.Errorf("external rule: command failed: %w: %s", err, stderr.String())
		}
		return Response{}, fmt.Errorf("external rule: command failed: %w", err)
	}
	if stdout.exceeded {
		return Response{}, fmt.Errorf("external rule: response exceeds 8388608 bytes")
	}
	var response Response
	decoder := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&response); err != nil {
		return Response{}, fmt.Errorf("external rule: decode response: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Response{}, fmt.Errorf("external rule: response contains trailing data")
		}
		return Response{}, fmt.Errorf("external rule: decode response: %w", err)
	}
	if response.ProtocolVersion != ProtocolVersion {
		return Response{}, fmt.Errorf("external rule: unsupported response protocol version %d", response.ProtocolVersion)
	}
	return response, nil
}

type limitedBuffer struct {
	bytes.Buffer
	limit    int
	exceeded bool
}

func (buffer *limitedBuffer) Write(value []byte) (int, error) {
	remaining := buffer.limit - buffer.Len()
	if remaining <= 0 {
		buffer.exceeded = true
		return len(value), nil
	}
	if len(value) > remaining {
		_, _ = buffer.Buffer.Write(value[:remaining])
		buffer.exceeded = true
		return len(value), nil
	}
	return buffer.Buffer.Write(value)
}
