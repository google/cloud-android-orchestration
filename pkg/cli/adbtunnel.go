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
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"

	client "github.com/google/cloud-android-orchestration/pkg/client"
	wclient "github.com/google/cloud-android-orchestration/pkg/webrtcclient"

	"github.com/hashicorp/go-multierror"
	"github.com/pion/webrtc/v3"
	"github.com/spf13/cobra"
)

const connectFlag = "connect"
const adbTunnelDir = "~/.cvdremote/adbtunnel"

type adbTunnelFlags struct {
	*subCommandFlags
	host string
}

type openADBTunnelFlags struct {
	*adbTunnelFlags
	connect bool
}

func newADBTunnelCommand(cfgFlags *configFlags, opts *subCommandOpts) *cobra.Command {
	adbTunnelFlags := &adbTunnelFlags{&subCommandFlags{cfgFlags, false}, ""}
	openFlags := &openADBTunnelFlags{adbTunnelFlags, false}
	open := &cobra.Command{
		Use:   "open",
		Short: "Opens an ADB tunnel.",
		RunE: func(c *cobra.Command, args []string) error {
			return runOpenADBTunnelCommand(openFlags, &command{c, &adbTunnelFlags.Verbose}, args, opts)
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
	addCommonSubcommandFlags(adbTunnel, adbTunnelFlags.subCommandFlags)
	adbTunnel.PersistentFlags().StringVar(&adbTunnelFlags.host, hostFlag, "", "Specifies the host")
	adbTunnel.AddCommand(open)
	adbTunnel.AddCommand(list)
	adbTunnel.AddCommand(close)
	return adbTunnel
}

func runOpenADBTunnelCommand(flags *openADBTunnelFlags, c *command, args []string, opts *subCommandOpts) error {
	adbPath := ""
	if flags.connect {
		path, err := exec.LookPath("adb")
		if err != nil {
			return fmt.Errorf("Can't connect adb: %w", err)
		}
		adbPath = path
	}
	service, err := opts.ServiceBuilder(flags.subCommandFlags, c.Command)
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
		controller, err := NewTunnelController(service, devProps, logger, &wg)

		if err != nil {
			merr = multierror.Append(merr, err)
			continue
		}

		ctrls = append(ctrls, controller)

		adbSerial := fmt.Sprintf("127.0.0.1:%d", controller.GetForwarderPort())
		if adbPath == "" {
			// Print the address to stdout if adb wasn't connected automatically
			c.Printf("%s: %s\n", device, adbSerial)
			continue
		}
		if err := ADBConnect(adbSerial, adbPath, c.ErrOrStderr()); err != nil {
			c.PrintErrf("Error connecting ADB to %q: %v", device, err)
			// Just print the error, don't add it to merr since the forwarding is working
			continue
		}
		c.PrintErrf("%s connected on: %s\n", device, adbSerial)
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
	dc        *webrtc.DataChannel
	listener  net.Listener
	conn      net.Conn
	port      int
	status    string
	statusMtx sync.Mutex
	logger    *log.Logger
}

const (
	ADBFwdInitializing = "initializing"
	ADBFwdReady        = "ready"
	ADBFwdConnected    = "connected"
	ADBFwdStopped      = "stopped"
	ADBFwdFailed       = "failed"
)

func (f *ADBForwarder) OnADBDataChannel(dc *webrtc.DataChannel) {
	f.dc = dc
	f.logger.Printf("ADB data channel to %q changed state: %v\n", f.device, dc.ReadyState())
	dc.OnOpen(func() {
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			if err := f.Send(msg.Data); err != nil {
				f.logger.Printf("Error writing to ADB server for %q: %v", f.device, err)
			}
		})
		dc.OnClose(func() {
			f.StopForwarding(ADBFwdFailed)
		})
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
	f.setStatus(ADBFwdReady)
	go f.acceptLoop()
}

func (f *ADBForwarder) StopForwarding(status string) {
	f.statusMtx.Lock()
	defer f.statusMtx.Unlock()

	if f.status == status {
		return
	}
	f.status = status
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
	Status     string `json:"status"`
}

func (f *ADBForwarder) StatusJSON() []byte {
	f.statusMtx.Lock()
	status := f.status
	f.statusMtx.Unlock()

	msg := StatusMsg{
		ServiceURL: f.serviceURL,
		Zone:       f.zone,
		Host:       f.host,
		Device:     f.device,
		Port:       f.port,
		Status:     status,
	}
	ret, err := json.Marshal(&msg)
	if err != nil {
		panic("Couldn't marshal status map")
	}
	return ret
}

func (f *ADBForwarder) setStatus(status string) {
	f.statusMtx.Lock()
	defer f.statusMtx.Unlock()
	f.status = status
}

func (f *ADBForwarder) acceptLoop() {
	f.statusMtx.Lock()
	if f.status != ADBFwdReady {
		// This function should always be called from StartForwarding
		f.logger.Printf("Forwarder accept loop started in wrong state: %s", f.state)
	}
	f.statusMtx.Unlock()

	defer f.listener.Close()
	for {
		conn, err := f.listener.Accept()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				f.logger.Printf("Error accepting connection on port %d: %v", f.port, err)
			}
			return
		}
		f.statusMtx.Lock()
		f.conn = conn
		if f.status != ADBFwdReady {
			// A different state means this loop should end
			f.conn.Close()
			f.statusMtx.Unlock()
			return
		}
		f.statusMtx.Unlock()
		f.logger.Printf("Connection received on port %d", f.port)
		f.recvLoop()
		f.statusMtx.Lock()
		if f.status != ADBFwdConnected {
			// A different state means this loop should end
			f.statusMtx.Unlock()
			return
		}
		f.status = ADBFwdReady
		f.statusMtx.Unlock()
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

func NewTunnelController(service client.Service, devProps deviceProperties,
	logger *log.Logger, wg *sync.WaitGroup) (*TunnelController, error) {
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
	control, err := createControlSocket(port)
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
	// Start control loop after forwarding loop to ensure the forwarder is in a valid state
	tc.StartControlLoop(wg)

	return tc, nil
}

func (tc *TunnelController) Stop() {
	// This will cause the forwarding loop to finish.
	tc.conn.Close()
	// This will cause the control loop to finish.
	tc.control.Close()
}

func (tc *TunnelController) StartControlLoop(wg *sync.WaitGroup) {
	wg.Add(1)
	go tc.controlSocketLoop(wg)
}

func (tc *TunnelController) GetForwarderPort() int {
	return tc.forwarder.port
}

func (tc *TunnelController) controlSocketLoop(wg *sync.WaitGroup) {
	defer wg.Done()
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

func createControlSocket(port int) (*net.UnixListener, error) {
	home := os.Getenv("HOME")
	dir := strings.ReplaceAll(adbTunnelDir, "~", home)
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

func ADBConnect(adbSerial string, adbPath string, errOut io.Writer) error {
	cmd := exec.Command(adbPath, "connect", adbSerial)
	// Make sure any adb errors are printed. Don't do the same for stdout: we'll print a
	// similar message with the device name.
	cmd.Stderr = errOut
	return cmd.Run()
}
