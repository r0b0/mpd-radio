package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"
)

type MpdData struct {
	client   *MpdClient
	Command  string
	Response map[string]string
	Binary   []byte
	Ok       string
	Unparsed []string
	Error    error
}

type CommandStruct struct {
	Command  string
	Response chan MpdData
}

func NewMpdData(client *MpdClient) MpdData {
	data := MpdData{}
	data.client = client
	data.Response = make(map[string]string)
	return data
}

var NotConnectedError = fmt.Errorf("not connected")

func (d *MpdData) Print() {
	if d.client == nil || d.client.logger == nil {
		return
	}
	d.client.logger.Debug("Command", "value", d.Command)
	responseValues := []slog.Attr{}
	for k, v := range d.Response {
		responseValues = append(responseValues, slog.String(k, v))
	}
	d.client.logger.LogAttrs(context.TODO(), slog.LevelDebug, "  Response", responseValues...)
	for _, line := range d.Unparsed {
		d.client.logger.Debug("  Unparsed line", "value", line)
	}
	for _, line := range d.Binary {
		d.client.logger.Debug("  Binary", "value", line)
	}
	d.client.logger.Debug("  Ok", "value", d.Ok)
}

type MpdClient struct {
	Address     string
	conn        io.ReadWriteCloser
	logger      *slog.Logger
	serveCancel context.CancelFunc
	lastUse     time.Time
	commands    chan CommandStruct
}

func NewMpdClient(ctx context.Context, host string, port string, parent *slog.Logger) (*MpdClient, error) {
	if port == "" {
		port = "6600"
	}
	address := net.JoinHostPort(host, port)
	client := MpdClient{address,
		nil,
		parent.With("player address", address),
		nil,
		time.Now(),
		make(chan CommandStruct)}
	err := client.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func (c *MpdClient) Connect(ctx context.Context) error {
	var err error
	c.conn, err = net.Dial("tcp", c.Address)
	if err != nil {
		return err
	}
	data, err := c.recv()
	if err != nil {
		return err
	}
	data.Print()
	child, cancel := context.WithCancel(ctx)
	c.serveCancel = cancel
	c.lastUse = time.Now()
	go c.Serve(child)
	return nil
}

func (c *MpdClient) Serve(ctx context.Context) {
	c.logger.Debug("Starting the serve goroutine")
	for {
		if c.conn != nil && time.Now().After(c.lastUse.Add(60*time.Second)) {
			c.logger.Info("No command for 60 seconds, disconnecting")
			c.Close()
			continue
		}
		resp := c.commandLow("ping")
		if resp.Error != nil {
			c.logger.Error("error when pinging", "error", resp.Error)
			c.Close()
		}
		select {
		case command := <-c.commands:
			resp := c.commandLow(command.Command)
			c.lastUse = time.Now()
			command.Response <- resp
		case <-ctx.Done():
			c.logger.Info("Closing the serve goroutine", "address", c.Address)
			return
		case <-time.After(30 * time.Second):
		}
	}
}

func (c *MpdClient) Close() {
	c.serveCancel()
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}

const MaxBinarySize = 1024 * 1024

func (c *MpdClient) recv() (MpdData, error) {
	data := NewMpdData(c)
	byteBuffer := make([]byte, 4096)
	n, err := c.conn.Read(byteBuffer)
	if err != nil {
		return data, err
	}
	readingBinary := 0
	lineStart := 0
	for i := range n {
		r := byteBuffer[i]
		if readingBinary == 0 {
			if r == '\n' {
				line := string(byteBuffer[lineStart:i])
				lineStart = i + 1
				after, has := strings.CutPrefix(line, "OK")
				if has {
					data.Ok = strings.TrimSpace(after)
					return data, nil
				}

				after, has = strings.CutPrefix(line, "ACK")
				if has {
					return data, fmt.Errorf("error from daemon: %s", after)
				}

				after, has = strings.CutPrefix(line, "Binary: ")
				if has {
					readingBinary, err = strconv.Atoi(after)
					if err != nil {
						return data, err
					}
					if readingBinary > MaxBinarySize {
						_ = c.conn.Close()
						c.conn = nil
						return data, fmt.Errorf("server requested binary size %d", readingBinary)
					}
				}

				k, v, has := strings.Cut(line, ": ")
				if has {
					data.Response[k] = v
				} else {
					data.Unparsed = append(data.Unparsed, line)
				}
			}
		} else {
			data.Binary = append(data.Binary, r)
			readingBinary--
		}
	}
	return data, fmt.Errorf("not enough data read from socket")
}

func (c *MpdClient) commandLow(command string) MpdData {
	c.logger.Debug("Running Command", "command", command)
	if c.conn == nil {
		return MpdData{Error: NotConnectedError}
	}
	_, err := c.conn.Write(fmt.Appendf(nil, "%s\n", command))
	if err != nil {
		return MpdData{Error: err}
	}
	resp, err := c.recv()
	resp.Error = err
	resp.Command = command
	return resp
}

func (c *MpdClient) Command(ctx context.Context, command string) MpdData {
	cmd := CommandStruct{
		Command:  command,
		Response: make(chan MpdData),
	}
	if c.conn == nil {
		err := c.Connect(context.Background())
		if err != nil {
			return MpdData{Error: fmt.Errorf("Failed to reconnect: %w", err)}
		}
	}
	if c.commands == nil {
		c.commands = make(chan CommandStruct)
		go c.Serve(context.Background()) // TODO what context?
	}
	select {
	case c.commands <- cmd:
		c.logger.Debug("successfully sent a command %s", command)
	case <-ctx.Done():
		return MpdData{Error: fmt.Errorf("Failed to send a command %s - context cancelled", command)}
	}
	resp := <-cmd.Response
	if errors.Is(resp.Error, NotConnectedError) {
		time.Sleep(1 * time.Second)
		err := c.Connect(ctx)
		if err != nil {
			return MpdData{Error: err}
		}
		return c.Command(ctx, command)
	} else {
		return resp
	}
}
