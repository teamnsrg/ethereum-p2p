package p2p

import (
	"fmt"
	"math/big"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/teamnsrg/go-ethereum/crypto"
	"github.com/teamnsrg/go-ethereum/log"
	"github.com/teamnsrg/go-ethereum/p2p/discover"
)

// Info represents a short summary of the information known about a known node.
type Info struct {
	mux sync.RWMutex

	RowID           uint64     `json:"RowID"`                     // Most recent row ID
	Keccak256Hash   string     `json:"keccak256Hash"`             // Keccak256 hash of node ID
	LastConnectedAt *time.Time `json:"lastConnectedAt,omitempty"` // Last time the node was connected
	IP              string     `json:"ip"`                        // IP address of the node
	TCPPort         uint16     `json:"tcpPort"`                   // TCP listening port for RLPx
	RemotePort      uint16     `json:"remotePort"`                // Remote TCP port of the most recent connection

	// DEVp2p Hello info
	P2PVersion   uint64     `json:"p2pVersion,omitempty"`   // DEVp2p protocol version
	ClientId     string     `json:"clientId,omitempty"`     // Name of the node, including client type, version, OS, custom data
	Caps         string     `json:"caps,omitempty"`         // Node's capabilities
	ListenPort   uint16     `json:"listenPort,omitempty"`   // Listening port reported in the node's DEVp2p Hello
	FirstHelloAt *time.Time `json:"firstHelloAt,omitempty"` // First time the node sent Hello
	LastHelloAt  *time.Time `json:"lastHelloAt,omitempty"`  // Last time the node sent Hello

	// Ethereum Status info
	ProtocolVersion uint64     `json:"protocolVersion,omitempty"` // Ethereum sub-protocol version
	NetworkId       uint64     `json:"networkId,omitempty"`       // Ethereum network ID
	FirstReceivedTd *big.Int   `json:"firstReceivedTd,omitempty"` // First reported total difficulty of the node's blockchain
	LastReceivedTd  *big.Int   `json:"lastReceivedTd,omitempty"`  // Last reported total difficulty of the node's blockchain
	BestHash        string     `json:"bestHash,omitempty"`        // Hex string of SHA3 hash of the node's best owned block
	GenesisHash     string     `json:"genesisHash,omitempty"`     // Hex string of SHA3 hash of the node's genesis block
	FirstStatusAt   *time.Time `json:"firstStatusAt,omitempty"`   // First time the node sent Status
	LastStatusAt    *time.Time `json:"lastStatusAt,omitempty"`    // Last time the node sent Status
	DAOForkSupport  bool       `json:"daoForkSupport"`            // Whether the node supports or opposes the DAO hard-fork
}

func (k *Info) Lock() {
	k.mux.Lock()
}

func (k *Info) Unlock() {
	k.mux.Unlock()
}

func (k *Info) RLock() {
	k.mux.RLock()
}

func (k *Info) RUnlock() {
	k.mux.RUnlock()
}

type knownNodeInfos struct {
	mux   sync.RWMutex
	infos map[discover.NodeID]*Info
}

func (k *knownNodeInfos) Lock() {
	k.mux.Lock()
}

func (k *knownNodeInfos) Unlock() {
	k.mux.Unlock()
}

func (k *knownNodeInfos) RLock() {
	k.mux.RLock()
}

func (k *knownNodeInfos) RUnlock() {
	k.mux.RUnlock()
}

func (k *knownNodeInfos) Infos() map[discover.NodeID]*Info {
	return k.infos
}

func (srv *Server) getNodeAddress(c *conn, receivedAt *time.Time) (*Info, bool, bool) {
	var (
		remoteIP   string
		remotePort uint16
		tcpPort    uint16
		dial       bool
		accept     bool
	)
	addrArr := strings.Split(c.fd.RemoteAddr().String(), ":")
	addrLen := len(addrArr)
	remoteIP = strings.Join(addrArr[:addrLen-1], ":")
	if p, err := strconv.ParseUint(addrArr[addrLen-1], 10, 16); err == nil {
		remotePort = uint16(p)
	}
	oldNodeInfo := srv.KnownNodeInfos.Infos()[c.id]

	var hash string
	if oldNodeInfo != nil {
		hash = oldNodeInfo.Keccak256Hash
		tcpPort = oldNodeInfo.TCPPort
	} else {
		hash = crypto.Keccak256Hash(c.id[:]).String()[2:]
	}
	// if inbound connection, resolve the node's listening port
	// otherwise, remotePort is the listening port
	if c.flags&inboundConn != 0 || c.flags&trustedConn != 0 {
		if tcpPort == 0 {
			newNode := srv.ntab.Resolve(c.id)
			// if the node address is resolved, set the tcpPort
			// otherwise, leave it as 0
			if newNode != nil {
				tcpPort = newNode.TCP
			}
		}
		accept = true
	} else {
		tcpPort = remotePort
		dial = true
	}
	newNodeInfo := &Info{
		Keccak256Hash: hash,
		LastHelloAt:   receivedAt,
		IP:            remoteIP,
		TCPPort:       tcpPort,
		RemotePort:    remotePort,
	}
	return newNodeInfo, dial, accept
}

func (srv *Server) storeNodeInfo(c *conn, receivedAt *time.Time, hs *protoHandshake) {
	// node address currentInfo
	newInfo, dial, accept := srv.getNodeAddress(c, receivedAt)
	id := hs.ID
	nodeid := id.String()
	if srv.addNodeMetaInfoStmt != nil {
		srv.addNodeMetaInfo(nodeid, newInfo.Keccak256Hash, dial, accept, false)
	}

	// DEVp2p Hello
	p2pVersion, clientId, capsArray, listenPort := hs.Version, hs.Name, hs.Caps, uint16(hs.ListenPort)
	caps := ""
	capsLen := len(capsArray)
	for i, c := range capsArray {
		caps += fmt.Sprintf("%s", c.String())
		if i < capsLen-1 {
			caps += ","
		}
	}
	clientId = strings.Replace(clientId, "'", "", -1)
	clientId = strings.Replace(clientId, "\"", "", -1)
	caps = strings.Replace(caps, "'", "", -1)
	caps = strings.Replace(caps, "\"", "", -1)

	newInfo.P2PVersion = p2pVersion
	newInfo.ClientId = clientId
	newInfo.Caps = caps
	newInfo.ListenPort = listenPort

	srv.KnownNodeInfos.Lock()
	defer srv.KnownNodeInfos.Unlock()
	if currentInfo, ok := srv.KnownNodeInfos.Infos()[id]; !ok {
		newInfo.FirstHelloAt = newInfo.LastHelloAt
		if srv.addNodeInfoStmt != nil {
			srv.addNodeInfo(&KnownNodeInfosWrapper{nodeid, newInfo})
		}
		if srv.GetRowIDStmt != nil {
			if rowID := srv.getRowID(nodeid); rowID > 0 {
				newInfo.RowID = rowID
			}
		}
		srv.KnownNodeInfos.Infos()[id] = newInfo

		// add the new node as a static node
		srv.addNewStatic(id, newInfo)
	} else {
		currentInfo.Lock()
		currentInfo.LastHelloAt = newInfo.LastHelloAt
		currentInfo.RemotePort = newInfo.RemotePort
		if nodeInfoChanged(currentInfo, newInfo) {
			currentInfo.IP = newInfo.IP
			currentInfo.TCPPort = newInfo.TCPPort
			currentInfo.P2PVersion = p2pVersion
			currentInfo.ClientId = clientId
			currentInfo.Caps = caps
			currentInfo.ListenPort = listenPort
			if srv.addNodeInfoStmt != nil {
				// TODO: check logic
				// in-memory entry should keep the Ethereum Status info
				// new entry to the mysql db should contain only the new address, DEVp2p info
				// let Ethereum protocol update the Status info, if available.
				srv.addNodeInfo(&KnownNodeInfosWrapper{nodeid, currentInfo})
			}
			if srv.GetRowIDStmt != nil {
				if rowID := srv.getRowID(nodeid); rowID > 0 {
					currentInfo.RowID = rowID
				}
			}
			// if the node's listening port changed
			// add it as a static node
			if currentInfo.TCPPort != newInfo.TCPPort {
				srv.addNewStatic(id, newInfo)
			}
		} else {
			if srv.updateNodeInfoStmt != nil {
				srv.updateNodeInfo(&KnownNodeInfosWrapper{nodeid, currentInfo})
			}
		}
		currentInfo.Unlock()
	}
}

func nodeInfoChanged(oldInfo *Info, newInfo *Info) bool {
	return oldInfo.IP != newInfo.IP || oldInfo.TCPPort != newInfo.TCPPort || oldInfo.P2PVersion != newInfo.P2PVersion ||
		oldInfo.ClientId != newInfo.ClientId || oldInfo.Caps != newInfo.Caps || oldInfo.ListenPort != newInfo.ListenPort
}

// During the initial node info loading process
// if a node seems to be listening (ie TCPPort != 0)
// add it as a static node
func (srv *Server) addInitialStatic(id discover.NodeID, nodeInfo *Info) {
	if nodeInfo.TCPPort != 0 {
		var ip net.IP
		if ip = net.ParseIP(nodeInfo.IP); ip == nil {
			log.Debug("Failed to add node to initial StaticNodes list", "node", fmt.Sprintf("enode://%s@%s:%d", id.String(), nodeInfo.IP, nodeInfo.TCPPort), "err", "failed to parse ip")
		} else {
			// Ensure the IP is 4 bytes long for IPv4 addresses.
			if ipv4 := ip.To4(); ipv4 != nil {
				ip = ipv4
			}
			log.Trace("Adding node to initial StaticNodes list", "node", fmt.Sprintf("enode://%s@%s:%d", id.String(), nodeInfo.IP, nodeInfo.TCPPort))
			srv.StaticNodes = append(srv.StaticNodes, discover.NewNode(id, ip, nodeInfo.TCPPort, nodeInfo.TCPPort))
		}
	}
}

// if a node seems to be listening (ie TCPPort != 0)
// add it as a static node
func (srv *Server) addNewStatic(id discover.NodeID, nodeInfo *Info) {
	if nodeInfo.TCPPort != 0 {
		var ip net.IP
		if ip = net.ParseIP(nodeInfo.IP); ip == nil {
			log.Debug("Failed to add static node", "node", fmt.Sprintf("enode://%s@%s:%d", id.String(), nodeInfo.IP, nodeInfo.TCPPort), "err", "failed to parse ip")
		} else {
			// Ensure the IP is 4 bytes long for IPv4 addresses.
			if ipv4 := ip.To4(); ipv4 != nil {
				ip = ipv4
			}
			srv.AddPeer(discover.NewNode(id, ip, nodeInfo.TCPPort, nodeInfo.TCPPort))
		}
	}
}

type KnownNodeInfosWrapper struct {
	NodeId string `json:"nodeid"` // Unique node identifier (also the encryption key)
	Info   *Info  `json:"info"`
}

// NodeInfo gathers and returns a collection of metadata known about the host.
func (srv *Server) KnownNodes() []*KnownNodeInfosWrapper {
	srv.KnownNodeInfos.Lock()
	defer srv.KnownNodeInfos.Unlock()
	infos := make([]*KnownNodeInfosWrapper, 0, len(srv.KnownNodeInfos.Infos()))
	for id, info := range srv.KnownNodeInfos.Infos() {
		nodeInfo := &KnownNodeInfosWrapper{
			id.String(),
			info,
		}
		infos = append(infos, nodeInfo)
	}
	// Sort the result array alphabetically by node identifier
	for i := 0; i < len(infos); i++ {
		for j := i + 1; j < len(infos); j++ {
			if infos[i].NodeId > infos[j].NodeId {
				infos[i], infos[j] = infos[j], infos[i]
			}
		}
	}
	return infos
}
