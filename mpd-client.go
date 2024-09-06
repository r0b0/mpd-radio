package main

import (
	"context"
	"fmt"
	"io"
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

func (d *MpdData) Print() {
	fmt.Printf("Original Command: %s\n", d.Command)
	for k, v := range d.Response {
		fmt.Printf("Received '%s': '%s'\n", k, v)
	}
	for _, line := range d.Unparsed {
		fmt.Printf("Received Unparsed line: '%s'\n", line)
	}
	for _, line := range d.Binary {
		fmt.Printf("Received Binary: '%v'\n", line)
	}
	fmt.Printf("Received Ok: %s\n", d.Ok)
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
			fmt.Printf("error when pinging: %v\n", err)
			return
		}
		select {
		case <-ctx.Done():
			fmt.Printf("Closing the pinger goroutine for %s", c.Address)
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
	fmt.Printf("Running Command %s\n", command)
	if c.conn == nil {
		return MpdData{}, fmt.Errorf("c not connected")
	}
	_, err := c.conn.Write([]byte(fmt.Sprintf("%s\n", command)))
	if err != nil {
		return MpdData{}, err
	}
	resp, err := c.recv()
	resp.Command = command
	return resp, err
}
