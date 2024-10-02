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
	Command  string
	Response map[string]string
	Binary   []byte
	Ok       string
	Unparsed []string
}

func NewMpdData() MpdData {
	data := MpdData{}
	data.Response = make(map[string]string)
	return data
}

var NotConnectedError = fmt.Errorf("not connected")

func (d *MpdData) Print() {
	slog.Debug("Command", "value", d.Command)
	for k, v := range d.Response {
		slog.Debug("  Response", "key", k, "value", v)
	}
	for _, line := range d.Unparsed {
		slog.Debug("  Unparsed line", "value", line)
	}
	for _, line := range d.Binary {
		slog.Debug("  Binary", "value", line)
	}
	slog.Debug("  Ok", "value", d.Ok)
}

type MpdClient struct {
	Address string
	conn    io.ReadWriteCloser
	mu      sync.Mutex
}

func NewMpdClient(ctx context.Context, host string, port string) (*MpdClient, error) {
	if port == "" {
		port = "6600"
	}
	client := MpdClient{net.JoinHostPort(host, port), nil, sync.Mutex{}}
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
	go c.Ping(ctx)
	return nil
}

func (c *MpdClient) Ping(ctx context.Context) {
	for {
		_, err := c.Command("ping")
		if err != nil {
			slog.Error("error when pinging", "error", err)
			_ = c.conn.Close()
			c.conn = nil
			return
		}
		select {
		case <-ctx.Done():
			slog.Info("Closing the pinger goroutine", "address", c.Address)
			_ = c.conn.Close()
			c.conn = nil
			return
		case <-time.After(30 * time.Second):
		}
	}
}

func (c *MpdClient) Close() {
	_ = c.conn.Close()
	c.conn = nil
}

const MaxBinarySize = 1024 * 1024

func (c *MpdClient) recv() (MpdData, error) {
	data := NewMpdData()
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
						c.Close()
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

func (c *MpdClient) Command(command string) (MpdData, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	slog.Debug("Running Command", "command", command)
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
