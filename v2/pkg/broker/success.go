package broker

type Message struct {
	CodeInt   int         `json:"code"`
	DataIface interface{} `json:"data"`
}
