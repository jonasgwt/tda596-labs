package commands

import (
	"bufio"
	"chord/node"
	"fmt"
	"os"
	"strings"
)

func CommandLoop(n *node.Node) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Chord node is operational. Use commands: 'Lookup <file>', 'StoreFile <file>', 'PrintState'.")
	for {
		fmt.Print("> ")
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		args := strings.SplitN(line, " ", 2)
		cmd := args[0]
		param := ""
		if len(args) > 1 {
			param = args[1]
		}

		switch cmd {
		case "Lookup":
			handleLookup(n, param)
		case "StoreFile":
			handleStoreFile(n, param)
		case "PrintState":
			n.PrintState()
		default:
			fmt.Println("Unknown command. Available commands: 'Lookup', 'StoreFile', 'PrintState'.")
		}
	}
}

func handleLookup(n *node.Node, fileName string) {
	successor, err := n.Lookup(fileName)
	if err != nil {
		fmt.Printf("Lookup failed: %v\n", err)
		return
	}
	fmt.Printf("File '%s' is managed by node: %s (%s)\n", fileName, successor.ID.Text(16), successor.Address)
}

func handleStoreFile(n *node.Node, filePath string) {
	if err := n.StoreFile(filePath); err != nil {
		fmt.Printf("StoreFile failed: %v\n", err)
		return
	}
	fmt.Printf("File '%s' stored successfully.\n", filePath)
}
