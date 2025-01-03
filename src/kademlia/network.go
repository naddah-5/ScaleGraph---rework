package kademlia

import (
	"errors"
	"fmt"
	"log"
	"sync"
)

type table struct {
	content map[[5]uint32]chan RPC
	sync.RWMutex
}

func NewTable() *table {
	ch := make(map[[5]uint32]chan RPC, 1024)
	return &table{
		content: ch,
	}
}

// Creates a RPC channel corresponding to the given id.
// Channel is entered into network table and returned.
// Returns an error if id is already in use.
func (table *table) Add(id [5]uint32) (chan RPC, error) {
	table.Lock()
	defer table.Unlock()

	_, exists := table.content[id]
	if exists {
		return make(chan RPC), errors.New("RPC id in use")
	}

	respChan := make(chan RPC)
	table.content[id] = respChan
	return respChan, nil
}

// Returns the matching RPC channel, or an error if there is no match.
func (table *table) RetrieveChan(id [5]uint32) (chan RPC, error) {
	table.Lock()
	defer table.Unlock()

	ch, ok := table.content[id]
	if !ok {
		return nil, errors.New("no matching RPC id")
	}
	delete(table.content, id)
	return ch, nil
}

// Removes entry with id from table.
func (table *table) DropChan(id [5]uint32) {
	table.Lock()
	defer table.Unlock()
	delete(table.content, id)
}

type Network struct {
	listener   chan RPC
	sender     chan RPC
	controller chan RPC
	serverIP   [4]byte
	masterNode Contact
	patience   int // Waiting time before giving up on reponse
	*table
}

// Returns a network pointer.
func NewNetwork(listener chan RPC, sender chan RPC, controller chan RPC, serverIP [4]byte, master Contact) *Network {
	newNetwork := Network{
		listener:   listener,
		sender:     sender,
		controller: controller,
		serverIP:   serverIP,
		masterNode: master,
		table:      NewTable(),
	}
	return &newNetwork
}

// Sends a RPC and creates a corresponding RPC id handle.
// Returns an error if the Response exceedes the timeout.
func (net *Network) Send(rpc RPC) (RPC, error) {
	log.Printf("sending %s to %v\tfrom node %v", rpc.cmd, rpc.receiver, rpc.sender.ID())
	if rpc.response {
		net.sender <- rpc
		return rpc, nil
	} else {
		rpc.id = RandomID()
		respChan, _ := net.Add(rpc.id)
		net.sender <- rpc
		res := <-respChan
		return res, nil
		// select {
		// case res := <-respChan:
		// 	log.Printf("received rpc\n%s", rpc.Display())
		// 	return res, nil
		// case <-time.After(TIMEOUT):
		// 	net.DropChan(rpc.ID)
		// 	break
		// }
	}
	// return rpc, errors.New("timeout")
}

// Start a listener on the network channel.
// Returns an error if the channel closes.
func (net *Network) Listen(node *Node) error {
	for {
		rpc, ok := <-net.listener
		if !ok {
			return errors.New("server down")
		}
		go net.route(node, rpc)
	}
}

// Routes the rpc to the appropriate components.
// If the rpc is a Response it tries to route it to that channel, otherwise routes it to the controller.
func (net *Network) route(node *Node, rpc RPC) {
	if rpc.response {
		respChan, err := net.RetrieveChan(rpc.id)
		if err != nil {
			errMSg := fmt.Sprintf("[ERROR] - possible time out\n error: %s", err.Error())
			log.Println(errMSg)
			return
		}
		go net.DropChan(rpc.id)
		respChan <- rpc
	} else {
		node.Handler(rpc)
	}
}
