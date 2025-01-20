package kademlia

import (
	"errors"
	"fmt"
	"log"
)

// Protocol handles the logic for sending RPC's

// Critical in order to reduce the risk of dead networks on start up.
// A dead network occurs when one or more nodes know of the network but is not known of by the network.
func (node *Node) Enter() {
	rpc := GenerateRPC(node.IP(), node.Contact)
	rpc.Enter()
	res, err := node.Send(rpc)
	if err != nil {
		log.Printf("%v - {ENTER} did not receive entry point", node.ID())
		return
	}
	if len(res.foundNodes) == 0 {
		log.Printf("[PANIC] - nil error prevented")
	}
	if res.foundNodes[0].IP() == [4]byte{0, 0, 0, 0} {
		log.Printf("%v - {ENTER} received illegal entry point", node.ID())
	}
	entryNode := res.foundNodes[0]
	branchNode := res.foundNodes[1]
	node.Ping(entryNode.IP())
	node.Ping(node.masterNode.IP())

	node.FindNode(node.Contact.ID())
	node.FindNode(branchNode.ID())
	node.FindNode(node.masterNode.ID())
}

// Logic for sending a ping RPC.
func (node *Node) Ping(address [4]byte) {
	rpc := GenerateRPC(address, node.Contact)
	rpc.Ping()
	res, err := node.Send(rpc)
	if err != nil {
		if node.debug {
			log.Printf("%v - [ERROR] RPC %v %s", node.ID(), rpc.id, err.Error())
		}
	}
	node.AddContact(res.sender)
}

func (node *Node) FindNode(target [5]uint32) []Contact {
	initNodes, _ := node.FindXClosest(REPLICATION, target)
	found := node.findNodeLoop(initNodes, target)
	return found
}

func (node *Node) findNodeLoop(prevContactList []Contact, target [5]uint32) []Contact {
	contactList := make([]Contact, 0, REPLICATION)
	respChan := make(chan []Contact, 64)

	for {
		// Launch parallel queries to initial nodes.
		for _, n := range prevContactList {
			rpc := GenerateRPC(n.IP(), node.Contact)
			rpc.FindNode(target)
			go node.findNodeQuery(rpc, respChan)
		}

		// Extract results from parallel query.
		for range prevContactList {
			resp, ok := <-respChan
			if ok {
				contactList = append(contactList, resp...)
			}
		}

		// Process the found contacts
		SortContactsByDistance(&contactList, target)
		RemoveDuplicateContacts(&contactList)
		if len(contactList) > CONCURRENCY {
			contactList = contactList[:REPLICATION]
		}

		if node.debug {
			pRes := fmt.Sprintf("found nodes:\n")
			for _, n := range contactList {
				pRes += fmt.Sprintf("%s\n", n.Display())
			}
			pRes += fmt.Sprintf("input nodes [DEBUG]:\n")
			for _, n := range prevContactList {
				pRes += fmt.Sprintf("%s\n", n.Display())
			}
			log.Printf(pRes)
		}

		if len(contactList) > 0 && len(prevContactList) > 0 {
			closer := CloserNode(contactList[0].ID(), prevContactList[0].ID(), target)
			if !closer {
				return contactList
			}
		} else if len(contactList) == 0 {
			return prevContactList
		}
		prevContactList = nil
		prevContactList = contactList
		contactList = make([]Contact, 0, REPLICATION)
	}
}

// Sends the given RPC and returns the reponse to the provided channel.
// If the RPC times out or returns an error, returns an empty contact.
// NOTE that you must assert the type of the result from respChan.
func (node *Node) findNodeQuery(rpc RPC, respChan chan []Contact) {
	resp, err := node.Send(rpc)
	if err != nil {
		if node.debug {
			log.Printf("[ERROR] - %s\nin node %v with rpc:\n%s\n", err.Error(), node.ID(), rpc.Display())
		}
		respChan <- resp.foundNodes
		return
	}
	for _, n := range resp.foundNodes {
		go node.Ping(n.IP())
	}
	respChan <- resp.foundNodes
	return

}

func (node *Node) StoreAccount(accID [5]uint32) {
	validators := node.FindNode(accID)
	for _, n := range validators {
		rpc := GenerateRPC(n.IP(), node.Contact)
		rpc.StoreAccount(accID)
	}
}

func (node *Node) FindAccount(accID [5]uint32) ([]Contact, error) {
	closeNodes := node.FindNode(accID)
	respChan := make(chan bool, REPLICATION)
	for _, n := range closeNodes {
		node.findAccountQuery(n.IP(), respChan, accID)
	}
	foundAccountNodes := 0
	for range len(closeNodes) {
		_, ok := <-respChan
		if !ok {
			continue
		} else {
			foundAccountNodes++
		}
	}
	if foundAccountNodes == REPLICATION {
		return closeNodes, nil
	} else {
		return closeNodes, errors.New(fmt.Sprintf("Failed to locate all nodes containing account: %v", accID))
	}
}

func (node *Node) findAccountQuery(target [4]byte, respChan chan bool, accID [5]uint32) {
	rpc := GenerateRPC(target, node.Contact)
	rpc.FindAccount(accID)
	res, err := node.Send(rpc)
	if err == nil {
		respChan <- res.findAccountSucc
	}
	respChan <- false
}
