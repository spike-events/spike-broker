package rids

import "fmt"

type spike struct{ Base }

var spikeImp *spike

// Spike internal events
func Spike() *spike {
	if spikeImp == nil {
		spikeImp = &spike{
			Base: NewRid("spike", "spike", "", 2),
		}
	}
	return spikeImp
}

func (r *spike) EventSocketConnected() Pattern {
	return r.NewMethod("User has IsPublic through Socket channel", "socket.connected").Event()
}

func (r *spike) EventSocketDisconnected(id ...fmt.Stringer) Pattern {
	return r.NewMethod("User has disconnected from Socket channel", "socket.disconnected.$Id", id...).Event()
}
