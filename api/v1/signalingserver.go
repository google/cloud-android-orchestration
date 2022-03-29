package v1

type ErrorMsg struct {
	Error string `json:"error"`
}

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

type InfraConfig struct {
	IceServers []IceServer `json:"ice_servers"`
}

type IceServer struct {
	URLs []string `json:"urls"`
}
