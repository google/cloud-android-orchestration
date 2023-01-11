package v1

type Error struct {
	Code     int    `json:"code"`
	ErrorMsg string `json:"error"`
}

type NewConnMsg struct {
	DeviceId string `json:"device_id"`
}

type NewConnReply struct {
	ConnId     string `json:"connection_id"`
	DeviceInfo any    `json:"device_info"`
}

type ForwardMsg struct {
	Payload any `json:"payload"`
}

type SServerResponse struct {
	Response   any
	StatusCode int
}

type InfraConfig struct {
	IceServers []IceServer `json:"ice_servers"`
}

type IceServer struct {
	URLs []string `json:"urls"`
}
