package v1

type NewConnMsg struct {
	DeviceId string `json:"device_id"`
}

type NewConnReply struct {
	ConnId     string      `json:"connection_id"`
	DeviceInfo interface{} `json:"device_info"`
}

type ForwardMsg struct {
	Payload interface{} `json:"payload"`
}

type SServerResponse struct {
	Response   interface{}
	StatusCode int
}
