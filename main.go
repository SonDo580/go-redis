package main

import (
	"fmt"
	"net"
	"strings"
)

func main() {
	// Create new server
	const PORT string = ":6379"
	listener, err := net.Listen("tcp", PORT)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Listening on port ", PORT)

	// Create/Open the AOF that stores RESP commands
	aof, err := NewAof("database.aof")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer aof.Close()

	// Listen for connections
	// (currently only accept 1 connection)
	conn, err := listener.Accept()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	for {
		// Read message from client
		resp := NewResp(conn)
		value, err := resp.Read()
		if err != nil {
			fmt.Println(err)
			return
		}

		if value.typ != RespTypeArray {
			fmt.Println("Invalid request, expected array")
			continue
		}

		if len(value.array) == 0 {
			fmt.Println("Invalid request, expected array length > 0")
			continue
		}

		command := strings.ToUpper(value.array[0].bulk)
		args := value.array[1:]

		writer := NewWriter(conn)

		handler, ok := Handlers[command]
		if !ok {
			fmt.Println("Invalid command: ", command)
			writer.Write(Value{typ: RespTypeString, str: ""})
			continue
		}

		// Append commands that modify database state to AOF
		if command == CmdSet || command == CmdHSet {
			aof.Write(value)
		}

		// Handle command and respond to client
		result := handler(args)
		writer.Write(result)
	}
}
