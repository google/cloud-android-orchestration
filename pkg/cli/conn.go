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
	"sync/atomic"
	"time"

	client "github.com/google/cloud-android-orchestration/pkg/client"
	wclient "github.com/google/cloud-android-orchestration/pkg/webrtcclient"

	"github.com/hashicorp/go-multierror"
	"github.com/pion/webrtc/v3"
)

type ForwarderState struct {
	Port  int    `json:"port"`
	State string `json:"state"`
}

type ConnStatus struct {
	ADB ForwarderState
}

type StatusCmdRes struct {
	CVD    RemoteCVDLocator
	Status ConnStatus
}

func ControlSocketName(_ RemoteCVDLocator, cs ConnStatus) string {
	// The canonical name is too long to use as a unix socket name, use the port
	// instead and use the canonical name to create a symlink to the socket.
	return fmt.Sprintf("%d.sock", cs.ADB.Port)
}

func EnsureConnDirsExist(controlDir string) error {
	if err := os.MkdirAll(controlDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", controlDir, err)
	}
	logsDir := logsDir(controlDir)
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}
	return nil
}

func DisconnectCVD(controlDir string, cvd RemoteCVDLocator, status ConnStatus) error {
	conn, err := net.Dial("unixpacket", fmt.Sprintf("%s/%s", controlDir, ControlSocketName(cvd, status)))
	if err != nil {
		return fmt.Errorf("failed to connect to %s/%s's agent: %w", cvd.Host, cvd.WebRTCDeviceID, err)
	}
	_, err = conn.Write([]byte(stopCmd))
	if err != nil {
		return fmt.Errorf("failed to send stop command to %s/%s: %w", cvd.Host, cvd.WebRTCDeviceID, err)
	}
	return nil
}

// Finds all existing connection agents. Returns the list of connection agents it was able
// to gather along with a multierror detailing the unreachable ones.
func listCVDConnections(controlDir string) (map[RemoteCVDLocator]ConnStatus, error) {
	var merr error
	statuses := make(map[RemoteCVDLocator]ConnStatus)
	entries, err := os.ReadDir(controlDir)
	if err != nil {
		return statuses, fmt.Errorf("error reading %s: %w", controlDir, err)
	}
	for _, file := range entries {
		if file.Type()&fs.ModeSocket == 0 {
			// Skip non socket files in the control directory.
			continue
		}
		path := fmt.Sprintf("%s/%s", controlDir, file.Name())
		conn, err := net.Dial("unixpacket", path)
		if err != nil {
			merr = multierror.Append(merr, fmt.Errorf("unable to contact connection agent: %w", err))
			continue
		}
		res, err := sendStatusCmd(conn)
		conn.Close()
		if err != nil {
			merr = multierror.Append(merr, err)
			continue
		}

		statuses[res.CVD] = res.Status
	}
	return statuses, merr
}

func listCVDConnectionsByHost(controlDir string, host string) (map[RemoteCVDLocator]ConnStatus, error) {
	l, err := listCVDConnections(controlDir)
	ret := make(map[RemoteCVDLocator]ConnStatus, 0)
	for cvd, status := range l {
		if host != cvd.Host {
			continue
		}
		ret[cvd] = status
	}
	return ret, err
}

type findOrConnRet struct {
	Status     ConnStatus
	Controller *ConnController
	Error      error
}

func FindOrConnect(controlDir string, cvd RemoteCVDLocator, service client.Service, localICEConfig *wclient.ICEConfig) (findOrConnRet, error) {
	statuses, err := listCVDConnectionsByHost(controlDir, cvd.Host)
	// Even with an error some connections may have been listed.
	if s, ok := statuses[cvd]; ok {
		return findOrConnRet{s, nil, err}, nil
	}
	// There is a race here since a connection could be created by a different process
	// after the checks were made above but before the socket was created below.
	// The likelihood of hitting that is very low though, and the effort required
	// to prevent it high, so we are choosing to live with it for the time being.
	controller, tErr := NewConnController(controlDir, service, cvd, localICEConfig)
	if tErr != nil {
		// This error is fatal, ingore any previous ones to avoid unnecessary noise.
		return findOrConnRet{}, fmt.Errorf("failed to create connection controller: %w", tErr)
	}

	return findOrConnRet{controller.Status(), controller, err}, nil
}

// Forwards messages between a local TCP server and a webrtc data channel.
type Forwarder struct {
	dc            *webrtc.DataChannel
	listener      net.Listener
	conn          net.Conn
	port          int
	state         int
	stateMtx      sync.Mutex
	logger        *log.Logger
	readyCh       chan struct{}
	readyChClosed atomic.Bool
}

func NewForwarder(logger *log.Logger) (*Forwarder, error) {
	// Bind the local socket before attempting to connect over WebRTC
	sock, err := bindTCPSocket()
	if err != nil {
		return nil, fmt.Errorf("failed to bind to local: %w", err)
	}
	port := sock.Addr().(*net.TCPAddr).Port

	f := &Forwarder{
		listener: sock,
		port:     port,
		logger:   logger,
		readyCh:  make(chan struct{}),
	}

	return f, nil
}

const (
	FwdInitializing = iota // 0: The initial state after creation
	FwdReady        = iota
	FwdConnected    = iota
	FwdStopped      = iota
	FwdFailed       = iota
)

func StateAsStr(state int) string {
	switch state {
	case FwdInitializing:
		return "initializing"
	case FwdReady:
		return "ready"
	case FwdConnected:
		return "connected"
	case FwdStopped:
		return "stopped"
	case FwdFailed:
		return "failed"
	}
	return "unknown"
}

func (f *Forwarder) OnDataChannel(dc *webrtc.DataChannel) {
	f.dc = dc
	dc.OnOpen(func() {
		f.logger.Printf("Data channel changed state: %v\n", dc.ReadyState())
		if set, prev := f.compareAndSwapState(FwdInitializing, FwdReady); !set {
			f.logger.Printf("Forwarding not started in unexpected state: %v", StateAsStr(prev))
			return
		}
		go f.acceptLoop()
		if f.readyChClosed.CompareAndSwap(false, true) {
			close(f.readyCh)
		}
	})
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		if err := f.Send(msg.Data); err != nil {
			f.logger.Printf("Error writing to socket: %v", err)
		}
	})
	dc.OnClose(func() {
		f.logger.Printf("Data channel changed state: %v\n", dc.ReadyState())
		f.StopForwarding(FwdFailed)
		if f.readyChClosed.CompareAndSwap(false, true) {
			close(f.readyCh)
		}
	})
}

func (f *Forwarder) StopForwarding(state int) {
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

func (f *Forwarder) Send(data []byte) error {
	if f.conn == nil {
		return fmt.Errorf("no connection yet on port %d", f.port)
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

func (f *Forwarder) State() ForwarderState {
	// Pass equal values to get the current state without changing it
	_, state := f.compareAndSwapState(-1, -1)

	return ForwarderState{
		Port:  f.port,
		State: StateAsStr(state),
	}
}

// Changes f.state to new if it had old. Returns whether the change was
// made and the old state.
func (f *Forwarder) compareAndSwapState(old, new int) (bool, int) {
	f.stateMtx.Lock()
	defer f.stateMtx.Unlock()
	if f.state == old {
		f.state = new
		return true, old
	}
	return false, f.state
}

func (f *Forwarder) setConnection(conn net.Conn) bool {
	f.stateMtx.Lock()
	defer f.stateMtx.Unlock()
	if f.state != FwdReady {
		return false
	}
	f.conn = conn
	f.state = FwdConnected
	return true
}

func (f *Forwarder) acceptLoop() {
	if changed, state := f.compareAndSwapState(FwdReady, FwdReady); !changed {
		f.logger.Printf("Forwarder accept loop started in wrong state: %s", StateAsStr(state))
		// This isn't necessarily an error, StopForwarding could have been called already
		return
	}

	f.logger.Printf("Listening on port %d", f.port)

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

		if changed, _ := f.compareAndSwapState(FwdConnected, FwdReady); !changed {
			// A different state means this loop should end
			return
		}
	}
}

func (f *Forwarder) recvLoop() {
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
			f.logger.Printf("Failed to send data to data channel from port %d: %v", f.port, err)
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

// Controls the webrtc connection maintained between the connection agent and a cvd.
// Implements the Observer interface for the webrtc client.
type ConnController struct {
	cvd          RemoteCVDLocator
	control      *net.UnixListener
	adbForwarder *Forwarder
	logger       *log.Logger
	webrtcConn   *wclient.Connection
}

func NewConnController(
	controlDir string,
	service client.Service,
	cvd RemoteCVDLocator,
	localICEConfig *wclient.ICEConfig) (*ConnController, error) {
	logger, err := createLogger(controlDir, cvd)
	if err != nil {
		return nil, err
	}
	logger.Printf("Connecting to %s in host %s", cvd.Name, cvd.Host)
	f, err := NewForwarder(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate ADB forwarder for %q: %w", cvd.WebRTCDeviceID, err)
	}

	tc := &ConnController{
		cvd:          cvd,
		adbForwarder: f,
		logger:       logger,
	}

	opts := client.ConnectWebRTCOpts{
		LocalICEConfig: localICEConfig,
	}
	conn, err := service.HostService(cvd.Host).ConnectWebRTC(cvd.WebRTCDeviceID, tc, logger.Writer(), opts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %q: %w", cvd.WebRTCDeviceID, err)
	}
	tc.webrtcConn = conn
	// TODO(jemoreira): close everything except the relevant data channels.

	// Wait for the ADB forwarder to be set up before connecting the ADB server.
	<-f.readyCh

	// Create the control socket as late as possible to reduce the chances of it
	// being left behind if the user interrupts the command.
	control, err := createControlSocket(controlDir, ControlSocketName(tc.cvd, tc.Status()))
	if err != nil {
		f.StopForwarding(FwdFailed)
		tc.webrtcConn.Close()
		return nil, fmt.Errorf("control socket creation failed for %q: %w", cvd.WebRTCDeviceID, err)
	}
	tc.control = control

	return tc, nil
}

func (tc *ConnController) OnADBDataChannel(dc *webrtc.DataChannel) {
	tc.logger.Printf("ADB data channel to %q changed state: %v\n", tc.cvd.WebRTCDeviceID, dc.ReadyState())
	tc.adbForwarder.OnDataChannel(dc)
}

func (tc *ConnController) OnError(err error) {
	tc.adbForwarder.StopForwarding(FwdFailed)
	tc.logger.Printf("Error on webrtc connection to %q: %v\n", tc.cvd.WebRTCDeviceID, err)
}

func (tc *ConnController) OnFailure() {
	tc.adbForwarder.StopForwarding(FwdFailed)
	tc.logger.Printf("WebRTC connection to %q set to failed state", tc.cvd.WebRTCDeviceID)
}

func (tc *ConnController) OnClose() {
	tc.adbForwarder.StopForwarding(FwdStopped)
	tc.logger.Printf("WebRTC connection to %q closed", tc.cvd.WebRTCDeviceID)
}

func (tc *ConnController) Stop() {
	tc.adbForwarder.StopForwarding(FwdStopped)
	// This will cause the control loop to finish.
	tc.control.Close()
}

func (tc *ConnController) ADBPort() int {
	return tc.adbForwarder.port
}

func (tc *ConnController) Status() ConnStatus {
	return ConnStatus{
		ADB: tc.adbForwarder.State(),
	}
}

func (tc *ConnController) Run() {
	if tc.control == nil {
		// It's ok to abort here: the control socket doesn't exist yet.
		tc.logger.Fatalf("The control socket has not been setup yet")
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
func (tc *ConnController) handleControlCommand(conn net.Conn) {
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
		reply := StatusCmdRes{
			CVD:    tc.cvd,
			Status: tc.Status(),
		}
		msg, err := json.Marshal(reply)
		if err != nil {
			tc.logger.Printf("Couldn't marshal status map: %v", err)
			return
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
func sendStatusCmd(conn net.Conn) (StatusCmdRes, error) {
	var msg StatusCmdRes
	// No need to write in a loop because it's a unixpacket socket, so all
	// messages are delivered in full or not at all.
	_, err := conn.Write([]byte(statusCmd))
	if err != nil {
		return msg, fmt.Errorf("failed to send status command: %w", err)
	}
	buff := make([]byte, 512)
	n, err := conn.Read(buff)
	if err != nil {
		return msg, fmt.Errorf("failed to read status command response: %w", err)
	}
	if err := json.Unmarshal(buff[:n], &msg); err != nil {
		return msg, fmt.Errorf("failed to parse status command response: %w", err)
	}
	return msg, nil
}

func bindTCPSocket() (net.Listener, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		return nil, fmt.Errorf("error listening on local TCP port: %w", err)
	}
	return listener, nil
}

func createControlSocket(dir, name string) (*net.UnixListener, error) {
	path := fmt.Sprintf("%s/%s", dir, name)

	// Use of "unixpacket" network is required to have message boundaries.
	control, err := net.ListenUnix("unixpacket", &net.UnixAddr{Name: path, Net: "unixpacket"})
	if err != nil {
		return nil, fmt.Errorf("failed to create control socket: %w", err)
	}

	control.SetUnlinkOnClose(true)

	return control, nil
}

func logsDir(controlDir string) string {
	return controlDir + "/logs"
}

func createLogger(controlDir string, dev RemoteCVDLocator) (*log.Logger, error) {
	logsDir := logsDir(controlDir)
	// The name looks like 123456_us-central1-c_cf-12345-12345_cvd-1.log
	path := fmt.Sprintf("%s/%d_%s_%s.log", logsDir, time.Now().Unix(), dev.Host, dev.WebRTCDeviceID)
	logsFile, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0660)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
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
		return 0, fmt.Errorf("failed to read logs dir: %w", err)
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

func loadICEConfigFromFile(path string) (*wclient.ICEConfig, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	config := struct {
		Config wclient.ICEConfig `json:"config"`
	}{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config.Config, nil
}
