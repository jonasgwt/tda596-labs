package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"os"

	"chord/commands"
	"chord/node"
)

func main() {
	address := flag.String("a", "", "The IP address that the Chord client will bind to and advertise to other nodes")
	port := flag.Int("p", 0, "The port that the Chord client will bind to and listen on")
	joinAddress := flag.String("ja", "", "The IP address of the machine running a Chord node to join")
	joinPort := flag.Int("jp", 0, "The port of the Chord node to join")
	ts := flag.Int("ts", 3000, "The time in milliseconds between invocations of 'stabilize'")
	tff := flag.Int("tff", 1000, "The time in milliseconds between invocations of 'fix fingers'")
	tcp := flag.Int("tcp", 3000, "The time in milliseconds between invocations of 'check predecessor'")
	flag.Parse()

	if *address == "" || *port == 0 {
		fmt.Println("Error: Both -a and -p must be specified.")
		flag.Usage()
		os.Exit(1)
	}

	nodeAddress := fmt.Sprintf("%s:%d", *address, *port)
	newNode := node.NewNode(nodeAddress)

	// Start the RPC server
	rpc.Register(newNode)
	listener, err := net.Listen("tcp", newNode.Address)
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		return
	}
	defer listener.Close()
	go rpc.Accept(listener)

	// Join an existing ring if specified
	if *joinAddress != "" && *joinPort != 0 {
		remoteAddress := fmt.Sprintf("%s:%d", *joinAddress, *joinPort)
		if err := newNode.Join(remoteAddress); err != nil {
			fmt.Printf("Join failed: %v\n", err)
			return
		}
	} else {
		fmt.Println("Creating a new Chord ring.")
		newNode.CreateRing()
	}

	// Start periodic tasks
	go newNode.StartStabilize(*ts)
	go newNode.StartFixFingers(*tff)
	go newNode.StartCheckPredecessor(*tcp)

	// Command loop
	commands.CommandLoop(newNode)
}
