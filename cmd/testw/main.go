package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/cloud-android-orchestration/pkg/webrtcclient"
	"github.com/pion/webrtc/v3"
)

type Observer struct {
	dc   *webrtc.DataChannel
	port int
}

func (o *Observer) OnADBDataChannel(dc *webrtc.DataChannel) {
	o.dc = dc
	fmt.Printf("Data Channel state: %v\n", dc.ReadyState())
	// Accept connections on the port only after the device is ready to accept messages
	dc.OnOpen(func() {
		fmt.Println("ADB Channel open -==-----====-===-=-====-===-")
		// TODO encapsulate in its own thing
		listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", o.port))
		if err != nil {
			// TODO
			panic(err)
		}
		defer listener.Close()
		var buffer [4096]byte
		for {
			conn, err := listener.Accept()
			if err != nil {
				panic(err)
			}
			dc.OnMessage(func(msg webrtc.DataChannelMessage) {
				// TODO what if the connection is lost?
				length := 0
				for length < len(msg.Data) {
					l, err := conn.Write(msg.Data[length:])
					if err != nil {
						panic(err)
					}
					length += l
				}
			})
			for {
				length, err := conn.Read(buffer[:])
				if err != nil {
					fmt.Printf("Error reading from conn: %v\n", err)
					break
				}
				if length == 0 {
					conn.Close()
					break
				}
				err = dc.Send(buffer[:length])
				if err != nil {
					panic(err)
				}
			}
		}
	})
}

func (o *Observer) OnError(err error) {
	fmt.Printf("Error on connection: %v\n", err)
}

type SignalHandler struct {
	connectionURL string
	client        *http.Client
}

func NewSignalHandler(baseUrl string, deviceId string) (*SignalHandler, error) {
	// TODO get infra config
	client := &http.Client{}
	// TODO use actual type, from the other github project
	var msg map[string]interface{}
	r, err := client.Post(fmt.Sprintf("%s/polled_connections", baseUrl), "application/json", strings.NewReader(fmt.Sprintf(`{"device_id": "%s"}`, deviceId)))
	if err != nil {
		return nil, fmt.Errorf("Error creating polled connection on server: %v", err)
	}
	err = json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode server's response: %v", err)
	}
	connId := msg["connection_id"].(string)
	fmt.Println("Connection id: ", connId)

	return &SignalHandler{
		connectionURL: fmt.Sprintf("%s/polled_connections/%s", baseUrl, connId),
		client:        client,
	}, nil
	// return nil, fmt.Errorf("Failed")
}

func (sh SignalHandler) Poll(sinkCh chan map[string]interface{}) {
	start := 0
	for {
		resp, err := sh.client.Get(fmt.Sprintf("%s/messages?start=%d", sh.connectionURL, start))
		if err != nil {
			// TODO interpret error, retry a couple times and then abort
			panic(err)
		}
		var messages []map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&messages)
		if err != nil {
			// TODO
			panic(err)
		}
		for _, message := range messages {
			// fmt.Printf("Device msg: %v\n", message)
			if message["message_type"] != "device_msg" {
				fmt.Println("unexpected message type!!!!!!!!!!!!!!!!!!!!!!!")
				continue
			}
			sinkCh <- message["payload"].(map[string]interface{})
			start++
		}
		// TODO back off
		time.Sleep(time.Second)
	}
}

func (sh SignalHandler) Forward(srcCh chan interface{}) {
	for {
		msg, open := <-srcCh
		if !open {
			break
		}
		bytes, err := json.Marshal(msg)
		if err != nil {
			// TODO handle
			panic(err)
		}
		// TODO use the actual JSON type
		_, err = sh.client.Post(
			fmt.Sprintf("%s/:forward", sh.connectionURL),
			"application/json",
			strings.NewReader(fmt.Sprintf(`{"message_type": "forward", "payload": %s}`, string(bytes))),
		)
		if err != nil {
			// TODO handle
			panic(err)
		}
	}
}

func (sh SignalHandler) InitHandling() webrtcclient.Signaling {
	signaling := webrtcclient.Signaling{
		SendCh: make(chan interface{}),
		RecvCh: make(chan map[string]interface{}),
		// TODO get servers from infra
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// TODO allow stopping
	go sh.Poll(signaling.RecvCh)
	go sh.Forward(signaling.SendCh)

	return signaling
}

func main() {
	observer := Observer{
		dc:   nil,
		port: 16520,
	}
	signalHandler, err := NewSignalHandler("http://127.0.0.1:1080", "cvd-1")
	if err != nil {
		panic(err)
	}
	signaling := signalHandler.InitHandling()
	_, err = webrtcclient.NewDeviceConnection(&signaling, &observer)
	if err != nil {
		panic(err)
	}
	fmt.Println("Connection established")
	// wait forever
	select {}
}
