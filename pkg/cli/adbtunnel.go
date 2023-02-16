// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	client "github.com/google/cloud-android-orchestration/pkg/client"
	wclient "github.com/google/cloud-android-orchestration/pkg/webrtcclient"

	"github.com/hashicorp/go-multierror"
	"github.com/pion/webrtc/v3"
)

// Starts an ADB agent process for each device. Waits for all started subprocesses
// to report the tunnel was successfully created or an error occurred. Returns a
// summary of errors reported by the ADB agents or nil if all succeeded. Some
// tunnels may have been established even if this function returns an error. Those
// are discoverable through listADBTunnels() after this function returns.
func openADBTunnels(devices []string, launchAgent func(string) int, reportTunnel func(string, int)) error {
	portChs := make([]chan int, len(devices))
	for i, device := range devices {
		portChs[i] = make(chan int)
		go func(ch chan int, device string) {
			defer close(ch)
			port := launchAgent(device)
			if port > 0 {
				ch <- port
			}
		}(portChs[i], device)
	}

	failed := 0
	var merr error
	for i, device := range devices {
		port, read := <-portChs[i]
		if !read {
			// Channel close without sending port number indicates a failure
			failed++
			continue
		}
		reportTunnel(device, port)

		if err := ADBConnect(port); err != nil {
			err = fmt.Errorf("Error connecting ADB to %q: %w\n", device, err)
			merr = multierror.Append(merr, err)
		}
	}
	if failed > 0 {
		err := fmt.Errorf("Failed to tunnel %d out of %d devices", failed, len(devices))
		merr = multierror.Append(merr, err)
	}
	return merr
}

func closeTunnel(controlDir string, dev ADBForwarderStatus) error {
	conn, err := net.Dial("unixpacket", fmt.Sprintf("%s/%d.sock", controlDir, dev.Port))
	if err != nil {
		return fmt.Errorf("Failed to connect to %s/%s's agent: %w", dev.Host, dev.Device, err)
	}
	_, err = conn.Write([]byte(stopCmd))
	if err != nil {
		return fmt.Errorf("Failed to send stop command to %s/%s: %w", dev.Host, dev.Device, err)
	}
	ADBDisconnect(dev.Port)
	return nil
}

// Finds all existing ADB tunnels. Returns the list of ADB tunnels it was able
// to gather along with a multierror detailing errors for the unreachable ones.
func listADBTunnels(controlDir string, zone, host string) ([]ADBForwarderStatus, error) {
	var merr error
	forwarders := make([]ADBForwarderStatus, 0)
	entries, err := os.ReadDir(controlDir)
	if err != nil {
		return forwarders, fmt.Errorf("Error reading %s: %w", controlDir, err)
	}
	for _, file := range entries {
		if file.Type()&fs.ModeSocket == 0 {
			// Skip non socket files in the control directory.
			continue
		}
		path := fmt.Sprintf("%s/%s", controlDir, file.Name())
		conn, err := net.Dial("unixpacket", path)
		if err != nil {
			merr = multierror.Append(merr, fmt.Errorf("Unable to contact tunnel agent: %w", err))
			continue
		}
		fwdr, err := sendStatusCmd(conn)
		conn.Close()
		if err != nil {
			merr = multierror.Append(merr, err)
			continue
		}

		if zone != "" && zone != fwdr.Zone {
			continue
		}

		if host != "" && fwdr.Host != host {
			continue
		}

		forwarders = append(forwarders, fwdr)
	}
	return forwarders, merr
}

type findOrCreateRet struct {
	Port       int
	Controller *TunnelController
	Error      error
}

func findOrCreateTunnel(controlDir string, devSpec DeviceSpec, service client.Service) (findOrCreateRet, error) {
	forwarders, err := listADBTunnels(controlDir, devSpec.Zone, devSpec.Host)
	// Even with an error some connections may have been listed.
	for _, fwd := range forwarders {
		if fwd.DeviceSpec == devSpec {
			return findOrCreateRet{fwd.Port, nil, err}, nil
		}
	}
	// There is a race here since a tunnel could be created by a different process
	// after the checks were made above but before the socket was created below.
	// The likelihood of hitting that is very low though, and the effort required
	// to prevent it high, so we are choosing to live with it for the time being.
	controller, tErr := NewTunnelController(controlDir, service, devSpec)
	if tErr != nil {
		// This error is fatal, ingore any previous ones to avoid unnecessary noise.
		return findOrCreateRet{}, fmt.Errorf("Failed to create tunnel controller: %w", tErr)
	}
	return findOrCreateRet{controller.Port(), controller, err}, nil
}

type DeviceSpec struct {
	ServiceURL string `json:"service_url"`
	Zone       string `json:"zone"`
	Host       string `json:"host"`
	Device     string `json:"device"`
}

// Forwards ADB messages between a local ADB server and a remote CVD.
// Implements the Observer interface for the webrtc client.
type ADBForwarder struct {
	DeviceSpec
	webrtcConn *wclient.Connection
	dc         *webrtc.DataChannel
	listener   net.Listener
	conn       net.Conn
	port       int
	state      int
	stateMtx   sync.Mutex
	logger     *log.Logger
	readyCh    chan struct{}
}

func NewADBForwarder(service client.Service, devSpec DeviceSpec, logger *log.Logger) (*ADBForwarder, error) {
	// Bind the two local sockets before attempting to connect over WebRTC
	tunnel, err := bindTCPSocket()
	if err != nil {
		return nil, fmt.Errorf("Failed to bind to local port for %q: %w", devSpec.Device, err)
	}
	port := tunnel.Addr().(*net.TCPAddr).Port

	f := &ADBForwarder{
		DeviceSpec: devSpec,
		listener:   tunnel,
		port:       port,
		logger:     logger,
		readyCh:    make(chan struct{}),
	}

	conn, err := service.ConnectWebRTC(f.Host, f.Device, f)
	if err != nil {
		return nil, fmt.Errorf("ADB tunnel creation failed for %q: %w", f.Device, err)
	}
	f.webrtcConn = conn
	// TODO(jemoreira): close everything except the ADB data channel.

	// Wait until the webrtc connection fails or the data channel opens.
	<-f.readyCh

	f.startForwarding()

	return f, nil
}

const (
	ADBFwdInitializing = iota // 0: The initial state after creation
	ADBFwdReady        = iota
	ADBFwdConnected    = iota
	ADBFwdStopped      = iota
	ADBFwdFailed       = iota
)

func StateAsStr(state int) string {
	switch state {
	case ADBFwdInitializing:
		return "initializing"
	case ADBFwdReady:
		return "ready"
	case ADBFwdConnected:
		return "connected"
	case ADBFwdStopped:
		return "stopped"
	case ADBFwdFailed:
		return "failed"
	default:
		panic(fmt.Sprintf("No known string representation for state: %d", state))
	}
}

func (f *ADBForwarder) OnADBDataChannel(dc *webrtc.DataChannel) {
	f.dc = dc
	f.logger.Printf("ADB data channel to %q changed state: %v\n", f.Device, dc.ReadyState())
	dc.OnOpen(func() {
		f.logger.Printf("ADB data channel to %q changed state: %v\n", f.Device, dc.ReadyState())
		close(f.readyCh)
	})
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		if err := f.Send(msg.Data); err != nil {
			f.logger.Printf("Error writing to ADB server for %q: %v", f.Device, err)
		}
	})
	dc.OnClose(func() {
		f.logger.Printf("ADB data channel to %q changed state: %v\n", f.Device, dc.ReadyState())
		f.StopForwarding(ADBFwdFailed)
	})
}

func (f *ADBForwarder) OnError(err error) {
	f.StopForwarding(ADBFwdFailed)
	f.logger.Printf("Error on webrtc connection to %q: %v\n", f.Device, err)
	// Unblock anyone waiting for the ADB channel to open.
	close(f.readyCh)
}

func (f *ADBForwarder) OnFailure() {
	f.StopForwarding(ADBFwdFailed)
	f.logger.Printf("WebRTC connection to %q set to failed state", f.Device)
}

func (f *ADBForwarder) OnClose() {
	f.StopForwarding(ADBFwdStopped)
	f.logger.Printf("WebRTC connection to %q closed", f.Device)
}

func (f *ADBForwarder) startForwarding() {
	if set, prev := f.compareAndSwapState(ADBFwdInitializing, ADBFwdReady); !set {
		f.logger.Printf("Forwarding not started in unexpected state: %v", StateAsStr(prev))
		return
	}
	go f.acceptLoop()
}

func (f *ADBForwarder) StopForwarding(state int) {
	f.stateMtx.Lock()
	defer f.stateMtx.Unlock()

	if f.state == state {
		return
	}
	f.state = state
	// Prevent future writes to the channel too.
	f.dc.Close()
	// f.listener is guaranteed to be non-nil at this point
	f.listener.Close()
	if f.conn != nil {
		f.conn.Close()
	}
}

func (f *ADBForwarder) Send(data []byte) error {
	if f.conn == nil {
		return fmt.Errorf("No connection yet on port %d", f.port)
	}
	// Once f.conn is not nil it's safe to use. The worst that can happen is that
	// we write to a closed connection which simply returns an error.
	length := 0
	for length < len(data) {
		l, err := f.conn.Write(data[length:])
		if err != nil {
			return err
		}
		length += l
	}
	return nil
}

type ADBForwarderStatus struct {
	DeviceSpec
	Port  int    `json:"port"`
	State string `json:"state"`
}

func (f *ADBForwarder) Status() ADBForwarderStatus {
	// Pass equal values to get the current state without changing it
	_, state := f.compareAndSwapState(-1, -1)

	return ADBForwarderStatus{
		DeviceSpec: DeviceSpec{
			ServiceURL: f.ServiceURL,
			Zone:       f.Zone,
			Host:       f.Host,
			Device:     f.Device,
		},
		Port:  f.port,
		State: StateAsStr(state),
	}
}

// Changes f.state to new if it had old. Returns whether the change was
// made and the old state.
func (f *ADBForwarder) compareAndSwapState(old, new int) (bool, int) {
	f.stateMtx.Lock()
	defer f.stateMtx.Unlock()
	if f.state == old {
		f.state = new
		return true, old
	}
	return false, f.state
}

func (f *ADBForwarder) setConnection(conn net.Conn) bool {
	f.stateMtx.Lock()
	defer f.stateMtx.Unlock()
	if f.state != ADBFwdReady {
		return false
	}
	f.conn = conn
	f.state = ADBFwdConnected
	return true
}

func (f *ADBForwarder) acceptLoop() {
	if changed, state := f.compareAndSwapState(ADBFwdReady, ADBFwdReady); !changed {
		f.logger.Printf("Forwarder accept loop started in wrong state: %s", StateAsStr(state))
		// This isn't necessarily an error, StopForwarding could have been called already
		return
	}

	defer f.listener.Close()
	for {
		conn, err := f.listener.Accept()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				f.logger.Printf("Error accepting connection on port %d: %v", f.port, err)
			}
			return
		}
		f.logger.Printf("Connection received on port %d", f.port)
		if !f.setConnection(conn) {
			// StopForwarding was called.
			conn.Close()
			return
		}

		f.recvLoop()

		if changed, _ := f.compareAndSwapState(ADBFwdConnected, ADBFwdReady); !changed {
			// A different state means this loop should end
			return
		}
	}
}

func (f *ADBForwarder) recvLoop() {
	defer f.conn.Close()
	var buffer [4096]byte
	for {
		length, err := f.conn.Read(buffer[:])
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				f.logger.Printf("Error receiving from port %d: %v", f.port, err)
			}
			return
		}
		err = f.dc.Send(buffer[:length])
		if err != nil {
			f.logger.Printf("Failed to send ADB data to %q: %v", f.Device, err)
			return
		}
	}
}

const (
	versionCmd = "version"
	statusCmd  = "status"
	stopCmd    = "stop"

	controlSocketCommsVersion = 1
)

type TunnelController struct {
	control   *net.UnixListener
	forwarder *ADBForwarder
	logger    *log.Logger
}

func NewTunnelController(controlDir string, service client.Service, devSpec DeviceSpec) (*TunnelController, error) {
	logger, err := createLogger(controlDir, devSpec)
	if err != nil {
		return nil, err
	}
	f, err := NewADBForwarder(service, devSpec, logger)
	if err != nil {
		return nil, err
	}

	// Create the control socket as late as possible to reduce the chances of it
	// being left behind if the user interrupts the command.
	control, err := createControlSocket(controlDir, f.port)
	if err != nil {
		f.StopForwarding(ADBFwdFailed)
		return nil, fmt.Errorf("Control socket creation failed for %q: %w", f.Device, err)
	}

	tc := &TunnelController{
		control:   control,
		forwarder: f,
		logger:    logger,
	}

	return tc, nil
}

func (tc *TunnelController) Stop() {
	tc.forwarder.StopForwarding(ADBFwdStopped)
	// This will cause the control loop to finish.
	tc.control.Close()
}

func (tc *TunnelController) Port() int {
	return tc.forwarder.port
}

func (tc *TunnelController) Run() {
	if tc.control == nil {
		panic("The control socket has not been setup yet")
	}
	for {
		conn, err := tc.control.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				// control socket closed, exit normally
				return
			}
			tc.logger.Printf("Error accepting connection on control socket: %v", err)
			continue
		}
		tc.handleControlCommand(conn)
		conn.Close()
	}
}

// TODO(jemoreira) add timeouts
func (tc *TunnelController) handleControlCommand(conn net.Conn) {
	buff := make([]byte, 100)
	n, err := conn.Read(buff)
	if err != nil {
		tc.logger.Printf("Error reading from control socket connection: %v", err)
		return
	}
	cmd := string(buff[:n])
	switch cmd {
	case versionCmd:
		_, err := conn.Write([]byte(fmt.Sprintf("%d", controlSocketCommsVersion)))
		if err != nil {
			tc.logger.Printf("Error writing to control socket connection: %v", err)
		}
	case statusCmd:
		status := tc.forwarder.Status()
		msg, err := json.Marshal(&status)
		if err != nil {
			panic(fmt.Sprintf("Couldn't marshal status map: %v", err))
		}
		_, err = conn.Write(msg)
		if err != nil {
			tc.logger.Printf("Error writing to control socket connection: %v", err)
		}
	case stopCmd:
		tc.Stop()
	default:
		tc.logger.Printf("Unknown command on control socket: %q", cmd)
	}
}

// The returned status is only valid if the error is nil
func sendStatusCmd(conn net.Conn) (ADBForwarderStatus, error) {
	var msg ADBForwarderStatus
	// No need to write in a loop because it's a unixpacket socket, so all
	// messages are delivered in full or not at all.
	_, err := conn.Write([]byte(statusCmd))
	if err != nil {
		return msg, fmt.Errorf("Failed to send status command: %w", err)
	}
	buff := make([]byte, 512)
	n, err := conn.Read(buff)
	if err != nil {
		return msg, fmt.Errorf("Failed to read status command response: %w", err)
	}
	if err := json.Unmarshal(buff[:n], &msg); err != nil {
		return msg, fmt.Errorf("Failed to parse status command response: %w", err)
	}
	return msg, nil
}

func bindTCPSocket() (net.Listener, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		return nil, fmt.Errorf("Error listening on local TCP port: %w", err)
	}
	return listener, nil
}

func createControlSocket(dir string, port int) (*net.UnixListener, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("Failed to create dir %s: %w", dir, err)
	}

	// The canonical name is too long to use as a unix socket name, use the port
	// instead and use the canonical name to create a symlink to the socket.
	name := fmt.Sprintf("%d.sock", port)
	path := fmt.Sprintf("%s/%s", dir, name)

	// Use of "unixpacket" network is required to have message boundaries.
	control, err := net.ListenUnix("unixpacket", &net.UnixAddr{Name: path, Net: "unixpacket"})
	if err != nil {
		fmt.Println(err)
		return nil, fmt.Errorf("Failed to create control socket: %w", err)
	}

	control.SetUnlinkOnClose(true)

	return control, nil
}

func ADBSendMsg(msg string) error {
	msg = fmt.Sprintf("%.4x%s", len(msg), msg)
	const ADBServerPort = 5037
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", ADBServerPort))
	if err != nil {
		return fmt.Errorf("Unable to contact ADB server: %w", err)
	}
	defer conn.Close()
	written := 0
	for written < len(msg) {
		n, err := conn.Write([]byte(msg[written:]))
		if err != nil {
			return fmt.Errorf("Error sending message to ADB server: %w", err)
		}
		written += n
	}
	return nil
}

func ADBConnect(port int) error {
	adbSerial := fmt.Sprintf("127.0.0.1:%d", port)
	msg := fmt.Sprintf("host:connect:%s", adbSerial)
	return ADBSendMsg(msg)
}

func ADBDisconnect(port int) error {
	adbSerial := fmt.Sprintf("127.0.0.1:%d", port)
	msg := fmt.Sprintf("host:disconnect:%s", adbSerial)
	return ADBSendMsg(msg)
}

func logsDir(controlDir string) string {
	return controlDir + "/logs"
}

func createLogger(controlDir string, dev DeviceSpec) (*log.Logger, error) {
	logsDir := logsDir(controlDir)
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("Failed to create logs dir: %w", err)
	}
	// The name looks like 123456_us-central1-c_cf-12345-12345_cvd-1.log
	path := fmt.Sprintf("%s/%d_%s_%s_%s.log", logsDir, time.Now().Unix(), dev.Zone, dev.Host, dev.Device)
	logsFile, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0660)
	if err != nil {
		return nil, fmt.Errorf("Failed to create log file: %w", err)
	}
	return log.New(logsFile, "", log.LstdFlags), nil
}

func maybeCleanOldLogs(controlDir string, inactiveTime time.Duration) (int, error) {
	if inactiveTime <= 0 {
		// A non-positive value causes to logs to never be deleted.
		return 0, nil
	}
	logsDir := logsDir(controlDir)
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		return 0, fmt.Errorf("Failed to read logs dir: %w", err)
	}
	var merr error
	now := time.Now()
	cnt := 0
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".log") {
			// skip non-log files
			continue
		}
		info, err := entry.Info()
		if err != nil {
			merr = multierror.Append(merr, err)
			continue
		}
		if now.Sub(info.ModTime()) > inactiveTime {
			if err := os.Remove(fmt.Sprintf("%s/%s", logsDir, entry.Name())); err != nil {
				merr = multierror.Append(merr, err)
			} else {
				cnt++
			}
		}
	}
	return cnt, merr
}
