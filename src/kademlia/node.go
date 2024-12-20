package kademlia

import (
	"time"
)

const (
	KEYSPACE      = 160 // the number of buckets
	KBUCKETVOLUME = 5   // K, number of contacts per bucket
	REPLICATION   = 3   // alpha
	CONCURRENCY   = 3
	PORT          = 8080
	DEBUG         = true
	POINT_DEBUG   = true
	TIMEOUT       = 10 * time.Second
)

type Node struct {
	Contact
	Network
	RoutingTable
	controller chan RPC // the channel for internal network, new rpc's are to be sent here for handling
}

func NewNode(id [5]uint32, ip [4]byte, listener chan RPC, sender chan RPC, serverIP [4]byte, masterNode Contact) *Node {
	controller := make(chan RPC)
	net := NewNetwork(listener, sender, controller, serverIP, masterNode)
	me := NewContact(ip, id)
	router := NewRoutingTable(id, KEYSPACE, KBUCKETVOLUME)
	return &Node{
		Contact: me,
		Network: *net,
		RoutingTable: *router,
	}
}

func (node *Node) Start() {
	go node.Network.Listen()
	rpc := GenerateRPC(node.Contact)
	rpc.Ping(node.masterNode.IP())
	go node.Send(rpc)
	time.Sleep(time.Millisecond * 10)
}
