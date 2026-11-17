// Package adb is a small client for the Android Debug Bridge server protocol
// (the adb server listening on 127.0.0.1:5037). It also locates or downloads
// Google's official platform-tools so the app stays a single portable file.
package adb

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// DefaultAddr is where the adb server listens.
func DefaultAddr() string {
	if a := os.Getenv("SMC_ADB_ADDR"); a != "" {
		return a
	}
	return "127.0.0.1:5037"
}

// Client talks to an adb server.
type Client struct {
	Addr    string
	Timeout time.Duration
}

// New returns a client for the default server address.
func New() *Client { return &Client{Addr: DefaultAddr(), Timeout: 20 * time.Second} }

// Device is one entry of "adb devices -l".
type Device struct {
	Serial  string `json:"serial"`
	State   string `json:"state"` // device | unauthorized | offline | recovery | sideload | bootloader | no permissions
	Product string `json:"product,omitempty"`
	Model   string `json:"model,omitempty"`
	Device  string `json:"device,omitempty"`
	USB     string `json:"usb,omitempty"`
}

func (c *Client) dial() (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", c.Addr, 3*time.Second)
	if err != nil {
		return nil, err
	}
	t := c.Timeout
	if t == 0 {
		t = 20 * time.Second
	}
	_ = conn.SetDeadline(time.Now().Add(t))
	return conn, nil
}

func send(conn net.Conn, req string) error {
	_, err := fmt.Fprintf(conn, "%04x%s", len(req), req)
	return err
}

func readStatus(conn net.Conn) error {
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return err
	}
	switch string(buf) {
	case "OKAY":
		return nil
	case "FAIL":
		msg, _ := readLenPrefixed(conn)
		return errors.New(msg)
	}
	return fmt.Errorf("adb: unexpected response %q", buf)
}

func readLenPrefixed(conn net.Conn) (string, error) {
	h := make([]byte, 4)
	if _, err := io.ReadFull(conn, h); err != nil {
		return "", err
	}
	n, err := strconv.ParseUint(string(h), 16, 32)
	if err != nil {
		return "", err
	}
	if n > 16<<20 {
		return "", fmt.Errorf("adb: reply too large (%d)", n)
	}
	b := make([]byte, n)
	_, err = io.ReadFull(conn, b)
	return string(b), err
}

func (c *Client) hostQuery(req string) (string, error) {
	conn, err := c.dial()
	if err != nil {
		return "", err
	}
	defer conn.Close()
	if err := send(conn, req); err != nil {
		return "", err
	}
	if err := readStatus(conn); err != nil {
		return "", err
	}
	return readLenPrefixed(conn)
}

// Version returns the adb server protocol version (proves the server runs).
func (c *Client) Version() (int, error) {
	s, err := c.hostQuery("host:version")
	if err != nil {
		return 0, err
	}
	v, err := strconv.ParseInt(s, 16, 32)
	return int(v), err
}

// Devices lists attached devices.
func (c *Client) Devices() ([]Device, error) {
	s, err := c.hostQuery("host:devices-l")
	if err != nil {
		return nil, err
	}
	var out []Device
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 2 {
			continue
		}
		d := Device{Serial: f[0], State: f[1]}
		rest := f[2:]
		if d.State == "no" && len(f) > 2 && f[2] == "permissions" {
			d.State = "no permissions"
		}
		for _, kv := range rest {
			k, v, ok := strings.Cut(kv, ":")
			if !ok {
				continue
			}
			switch k {
			case "product":
				d.Product = v
			case "model":
				d.Model = strings.ReplaceAll(v, "_", " ")
			case "device":
				d.Device = v
			case "usb":
				d.USB = v
			}
		}
		out = append(out, d)
	}
	return out, nil
}

// Shell runs a shell command on the device and returns its output.
func (c *Client) Shell(serial, cmd string) (string, error) {
	conn, err := c.dial()
	if err != nil {
		return "", err
	}
	defer conn.Close()
	if err := send(conn, "host:transport:"+serial); err != nil {
		return "", err
	}
	if err := readStatus(conn); err != nil {
		return "", err
	}
	if err := send(conn, "shell:"+cmd); err != nil {
		return "", err
	}
	if err := readStatus(conn); err != nil {
		return "", err
	}
	b, err := io.ReadAll(io.LimitReader(conn, 32<<20))
	if err != nil && len(b) == 0 {
		return "", err
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n"), nil
}
