package broker

type Message struct {
	CodeInt   int     `json:"code"`
	DataIface RawData `json:"data"`
}
