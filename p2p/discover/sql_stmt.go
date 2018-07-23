package discover

import (
	"time"

	"github.com/teamnsrg/ethereum-p2p/crypto"
)

func (t *udp) queueNeighbors(neighbors []rpcNode, currentTime time.Time) {
	unixTime := float64(currentTime.UnixNano()) / 1e9
	for _, n := range neighbors {
		t.neighborChan <- []interface{}{
			n.ID.String(),
			crypto.Keccak256Hash(n.ID[:]).String()[2:],
			n.IP.String(),
			n.TCP,
			n.UDP,
			unixTime,
			unixTime,
		}
	}
}
