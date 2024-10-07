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
	"sync"
	"time"
)

type MpdData struct {
	client   *MpdClient
	Command  string
	Response map[string]string
	Binary   []byte
	Ok       string
	Unparsed []string
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
	d.client.logger.LogAttrs(nil, slog.LevelDebug, "  Response", responseValues...)
	for _, line := range d.Unparsed {
		d.client.logger.Debug("  Unparsed line", "value", line)
	}
	for _, line := range d.Binary {
		d.client.logger.Debug("  Binary", "value", line)
	}
	d.client.logger.Debug("  Ok", "value", d.Ok)
}

type MpdClient struct {
	Address string
	conn    io.ReadWriteCloser
	lastUse time.Time
	mu      sync.Mutex
	logger  *slog.Logger
}

func NewMpdClient(ctx context.Context, host string, port string, parent *slog.Logger) (*MpdClient, error) {
	if port == "" {
		port = "6600"
	}
	address := net.JoinHostPort(host, port)
	client := MpdClient{address,
		nil,
		time.Now(),
		sync.Mutex{},
		parent.With("player address", address)}
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
	c.lastUse = time.Now()
	go c.Ping(ctx)
	return nil
}

func (c *MpdClient) Ping(ctx context.Context) {
	for {
		if time.Now().After(c.lastUse.Add(60 * time.Second)) {
			c.logger.Info("No command for 60 seconds, disconnecting")
			c.Close()
			return
		}
		_, err := c.commandLow("ping")
		if err != nil {
			c.logger.Error("error when pinging", "error", err)
			c.Close()
			return
		}
		select {
		case <-ctx.Done():
			c.logger.Info("Closing the pinger goroutine", "address", c.Address)
			c.Close()
			return
		case <-time.After(30 * time.Second):
		}
	}
}

func (c *MpdClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.Close()
	c.conn = nil
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
	for i := 0; i < n; i++ {
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

func (c *MpdClient) commandLow(command string) (MpdData, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.logger.Debug("Running Command", "command", command)
	if c.conn == nil {
		return MpdData{}, NotConnectedError
	}
	_, err := c.conn.Write([]byte(fmt.Sprintf("%s\n", command)))
	if err != nil {
		return MpdData{}, err
	}
	resp, err := c.recv()
	resp.Command = command
	return resp, err
}

func (c *MpdClient) Command(command string) (MpdData, error) {
	c.lastUse = time.Now()
	return c.commandLow(command)
}

func (c *MpdClient) CommandOrReconnect(ctx context.Context, command string) (MpdData, error) {
	resp, err := c.Command(command)
	if errors.Is(err, NotConnectedError) {
		time.Sleep(1 * time.Second)
		err := c.Connect(ctx)
		if err != nil {
			return MpdData{}, err
		}
		return c.Command(command)
	} else {
		return resp, err
	}
}
