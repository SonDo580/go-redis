package main

import (
	"fmt"
	"net"
	"strings"
)

func main() {
	// Start a TCP listener
	const PORT string = ":6379"
	listener, err := net.Listen("tcp", PORT)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Listening on port ", PORT)

	// Load the AOF
	aof, err := NewAof("database.aof")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer aof.Close()

	// Restore state
	aof.Read(func(value Value) {
		command := strings.ToUpper(value.array[0].bulk)
		args := value.array[1:]

		handler, ok := Handlers[command]
		if !ok {
			fmt.Println("Invalid command: ", command)
			return
		}
		handler(args)
	})

	// Accept connections (goroutine-per-client)
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		go handleClient(conn, aof)
	}

}

func handleClient(conn net.Conn, aof *Aof) {
	defer conn.Close()

	resp := NewResp(conn)
	writer := NewWriter(conn)

	for {
		value, err := resp.Read()
		if err != nil {
			fmt.Println(err)
			return
		}

		if value.typ != RespTypeArray || len(value.array) == 0 {
			writer.Write(Value{
				typ: RespTypeError,
				str: "ERR invalid request: expected non-empty array",
			})
			continue
		}

		command := strings.ToUpper(value.array[0].bulk)
		args := value.array[1:]

		handler, ok := Handlers[command]
		if !ok {
			writer.Write(Value{
				typ: RespTypeError,
				str: fmt.Sprintf("ERR unknown command '%s'", command),
			})
			continue
		}

		// Append mutating commands to AOF
		if command == CmdSet || command == CmdHSet {
			aof.Write(value)
		}

		result := handler(args)
		writer.Write(result)
	}
}
