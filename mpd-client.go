package main

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

type MpdData struct {
	response map[string]string
	binary   []byte
	ok       string
	unparsed []string
}

func NewMpdData() MpdData {
	data := MpdData{}
	data.response = make(map[string]string)
	return data
}

func (data *MpdData) Print() {
	for k, v := range data.response {
		fmt.Printf("Received '%s': '%s'\n", k, v)
	}
	for _, line := range data.unparsed {
		fmt.Printf("Received unparsed line: '%s'\n", line)
	}
	for _, line := range data.binary {
		fmt.Printf("Received binary: '%v'\n", line)
	}
	fmt.Printf("Received ok: %s\n", data.ok)
}

type MpdClient struct {
	address string
	conn    io.ReadWriteCloser
}

func (client *MpdClient) Close() {
	_ = client.conn.Close()
}

func Connect(host string, port string) (MpdClient, error) {
	if port == "" {
		port = "6600"
	}
	client := MpdClient{net.JoinHostPort(host, port), nil}
	var err error
	client.conn, err = net.Dial("tcp", client.address)
	if err != nil {
		return client, err
	}
	data, err := recv(client.conn)
	if err != nil {
		return client, err
	}
	data.Print()
	return client, nil
}

func mpdmain() {
	address := "pi.lamac.cc"
	conn, err := Connect(address, "")
	if err != nil {
		fmt.Println("Failed to connect: ", err)
		return
	}
	defer conn.Close()

	data, err := conn.Command("currentsong")
	if err != nil {
		fmt.Println("Failed to get current song: ", err)
		return
	}
	data.Print()

	/*
		data, err = command(conn, "idle")
		if err != nil {
		fmt.Println("Failed to idle: ", err)
		return
		}
		data.Print()
	*/
}

func recv(conn io.ReadWriter) (MpdData, error) {
	data := NewMpdData()
	byteBuffer := make([]byte, 4096)
	n, err := conn.Read(byteBuffer)
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
					data.ok = strings.TrimSpace(after)
					return data, nil
				}

				after, has = strings.CutPrefix(line, "ACK")
				if has {
					return data, fmt.Errorf("error from daemon: %s", after)
				}

				after, has = strings.CutPrefix(line, "binary: ")
				if has {
					readingBinary, err = strconv.Atoi(after)
					if err != nil {
						return data, err
					}
				}

				k, v, has := strings.Cut(line, ": ")
				if has {
					data.response[k] = v
				} else {
					data.unparsed = append(data.unparsed, line)
				}
			}
		} else {
			data.binary = append(data.binary, r)
			readingBinary--
		}
	}
	return data, fmt.Errorf("not enough data read from socket")
}

func (client *MpdClient) Command(command string) (MpdData, error) {
	fmt.Printf("Running command %s\n", command)
	_, err := client.conn.Write([]byte(fmt.Sprintf("%s\n", command)))
	if err != nil {
		return MpdData{}, err
	}
	return recv(client.conn)
}
