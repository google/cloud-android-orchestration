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
	"log"
	"net"
	"os"
	"strings"
	"sync"

	client "github.com/google/cloud-android-orchestration/pkg/client"
	wclient "github.com/google/cloud-android-orchestration/pkg/webrtcclient"

	"github.com/hashicorp/go-multierror"
	"github.com/pion/webrtc/v3"
	"github.com/spf13/cobra"
)

const connectFlag = "connect"

type ADBTunnelFlags struct {
	*CommonSubcmdFlags
	host string
}

type OpenADBTunnelFlags struct {
	*ADBTunnelFlags
	connect bool
}

func newADBTunnelCommand(opts *subCommandOpts) *cobra.Command {
	adbTunnelFlags := &ADBTunnelFlags{&CommonSubcmdFlags{opts.RootFlags, false}, ""}
	openFlags := &OpenADBTunnelFlags{adbTunnelFlags, false}
	open := &cobra.Command{
		Use:   "open",
		Short: "Opens an ADB tunnel.",
		RunE: func(c *cobra.Command, args []string) error {
			return openADBTunnel(openFlags, &command{c, &adbTunnelFlags.Verbose}, args, opts)
		},
	}
	open.PersistentFlags().BoolVarP(
		&openFlags.connect, connectFlag,
		"c",
		false,
		"Issue the `adb connect` command after the tunnel is open")
	list := &cobra.Command{
		Use:   "list",
		Short: "Lists open ADB tunnels.",
		RunE:  notImplementedCommand,
	}
	close := &cobra.Command{
		Use:   "close <foo> <bar> <baz>",
		Short: "Close ADB tunnels.",
		RunE:  notImplementedCommand,
	}
	adbTunnel := &cobra.Command{
		Use:   "adbtunnel",
		Short: "Work with ADB tunnels",
	}
	addCommonSubcommandFlags(adbTunnel, adbTunnelFlags.CommonSubcmdFlags)
	adbTunnel.PersistentFlags().StringVar(&adbTunnelFlags.host, hostFlag, "", "Specifies the host")
	adbTunnel.AddCommand(open)
	adbTunnel.AddCommand(list)
	adbTunnel.AddCommand(close)
	return adbTunnel
}

func openADBTunnel(flags *OpenADBTunnelFlags, c *command, args []string, opts *subCommandOpts) error {
	service, err := opts.ServiceBuilder(flags.CommonSubcmdFlags, c.Command)
	if err != nil {
		return err
	}

	var merr error
	var ctrls []*TunnelController
	var wg sync.WaitGroup

	for _, device := range args {
		logger := log.New(c.ErrOrStderr(), "", log.LstdFlags)
		devProps := deviceProperties{
			serviceURL: flags.ServiceURL,
			zone:       flags.Zone,
			host:       flags.host,
			device:     device,
		}
		controller, err := NewTunnelController(
			opts.InitialConfig.ADBControlDir, service, devProps, logger)

		if err != nil {
			merr = multierror.Append(merr, err)
			continue
		}

		wg.Add(1)
		go func() {
			controller.Run()
			wg.Done()
		}()
		ctrls = append(ctrls, controller)

		adbSerial := fmt.Sprintf("127.0.0.1:%d", controller.ForwarderPort())
		if flags.connect {
			if err := ADBConnect(adbSerial); err != nil {
				c.PrintErrf("Error connecting ADB to %q (%s): %v", adbSerial, device, err)
				merr = multierror.Append(merr, err)
			} else {
				c.PrintErrf("%s connected on: %s\n", device, adbSerial)
			}
		} else {
			c.Printf("%s: %s\n", device, adbSerial)
		}
	}

	// Wait for all controllers to stop
	wg.Wait()

	return merr
}

type deviceProperties struct {
	serviceURL string
	zone       string
	host       string
	device     string
}

// Forwards ADB messages between a local ADB server and a remote CVD.
// Implements the Observer interface for the webrtc client.
type ADBForwarder struct {
	deviceProperties
	dc       *webrtc.DataChannel
	listener net.Listener
	conn     net.Conn
	port     int
	state    int
	stateMtx sync.Mutex
	logger   *log.Logger
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
	f.logger.Printf("ADB data channel to %q changed state: %v\n", f.device, dc.ReadyState())
	dc.OnOpen(func() {
		f.logger.Printf("ADB data channel to %q changed state: %v\n", f.device, dc.ReadyState())
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			if err := f.Send(msg.Data); err != nil {
				f.logger.Printf("Error writing to ADB server for %q: %v", f.device, err)
			}
		})
	})
	dc.OnClose(func() {
		f.StopForwarding(ADBFwdFailed)
	})
}

func (f *ADBForwarder) OnError(err error) {
	f.StopForwarding(ADBFwdFailed)
	f.logger.Printf("Error on webrtc connection to %q: %v\n", f.device, err)
}

func (f *ADBForwarder) OnFailure() {
	f.StopForwarding(ADBFwdFailed)
	f.logger.Printf("WebRTC connection to %q set to failed state", f.device)
}

func (f *ADBForwarder) OnClose() {
	f.StopForwarding(ADBFwdStopped)
	f.logger.Printf("WebRTC connection to %q closed", f.device)
}

func (f *ADBForwarder) StartForwarding() {
	set, prev := f.compareAndSwapState(ADBFwdInitializing, ADBFwdReady)
	if !set {
		panic("StartForwarding called in wrong state: " + StateAsStr(prev))
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

type StatusMsg struct {
	ServiceURL string `json:"service_url"`
	Zone       string `json:"zone"`
	Host       string `json:"host"`
	Device     string `json:"device"`
	Port       int    `json:"port"`
	State      string `json:"state"`
}

func (f *ADBForwarder) StatusJSON() []byte {
	// Pass equal values to get the current state without chaning it
	_, state := f.compareAndSwapState(-1, -1)

	msg := StatusMsg{
		ServiceURL: f.serviceURL,
		Zone:       f.zone,
		Host:       f.host,
		Device:     f.device,
		Port:       f.port,
		State:      StateAsStr(state),
	}
	ret, err := json.Marshal(&msg)
	if err != nil {
		panic(fmt.Sprintf("Couldn't marshal status map: %v", err))
	}
	return ret
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
			f.logger.Printf("Failed to send ADB data to %q: %v", f.device, err)
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
	conn      *wclient.Connection
	forwarder *ADBForwarder
	logger    *log.Logger
}

func NewTunnelController(tunnelDir string, service client.Service, devProps deviceProperties,
	logger *log.Logger) (*TunnelController, error) {
	// Bind the two local sockets before attempting to connect over WebRTC
	tunnel, err := bindTCPSocket()
	if err != nil {
		return nil, fmt.Errorf("Failed to bind to local port for %q: %w", devProps.device, err)
	}
	port := tunnel.Addr().(*net.TCPAddr).Port

	f := &ADBForwarder{
		deviceProperties: devProps,
		listener:         tunnel,
		port:             port,
		logger:           logger,
	}

	conn, err := service.ConnectWebRTC(f.host, f.device, f)
	if err != nil {
		return nil, fmt.Errorf("ADB tunnel creation failed for %q: %w", f.device, err)
	}
	// TODO(jemoreira): close everything except the ADB data channel.

	// Create the control socket as late as possible to reduce the chances of it
	// being left behind if the user interrupts the command.
	control, err := createControlSocket(tunnelDir, port)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("Control socket creation failed for %q: %w", f.device, err)
	}

	tc := &TunnelController{
		control:   control,
		conn:      conn,
		forwarder: f,
		logger:    logger,
	}

	tc.forwarder.StartForwarding()

	return tc, nil
}

func (tc *TunnelController) Stop() {
	// This will cause the forwarding loop to finish.
	tc.conn.Close()
	// This will cause the control loop to finish.
	tc.control.Close()
}

func (tc *TunnelController) ForwarderPort() int {
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
		_, err := conn.Write(tc.forwarder.StatusJSON())
		if err != nil {
			tc.logger.Printf("Error writing to control socket connection: %v", err)
		}
	case stopCmd:
		tc.Stop()
	default:
		tc.logger.Printf("Unknown command on control socket: %q", cmd)
	}
}

func bindTCPSocket() (net.Listener, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		return nil, fmt.Errorf("Error listening on local TCP port: %w", err)
	}
	return listener, nil
}

func createControlSocket(dir string, port int) (*net.UnixListener, error) {
	home := os.Getenv("HOME")
	dir = strings.ReplaceAll(dir, "~", home)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("Failed to create dir %s: %w", dir, err)
	}

	// The canonical name is too long to use as a unix socket name, use the port
	// instead and use the canonical name to create a symlink to the socket.
	name := fmt.Sprintf("%d", port)
	path := fmt.Sprintf("%s/%s", dir, name)

	control, err := net.ListenUnix("unixpacket", &net.UnixAddr{Name: path, Net: "unixpacket"})
	if err != nil {
		fmt.Println(err)
		return nil, fmt.Errorf("Failed to create control socket: %w", err)
	}

	control.SetUnlinkOnClose(true)

	return control, nil
}

func ADBConnect(adbSerial string) error {
	const ADBServerPort = 5037
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", ADBServerPort))
	if err != nil {
		return fmt.Errorf("Unable to contact ADB server: %w", err)
	}
	defer conn.Close()
	msg := fmt.Sprintf("host:connect:%s", adbSerial)
	msg = fmt.Sprintf("%.4x%s", len(msg), msg)
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
