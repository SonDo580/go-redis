package main

import "sync"

const (
	CmdPing = "PING"
	CmdEcho = "ECHO"
	CmdSet  = "SET"
	CmdGet  = "GET"
	CmdHSet = "HSET"
	CmdHGet = "HGET"
)

// Map commands to handlers
var Handlers = map[string]func([]Value) Value{
	CmdPing: ping,
	CmdEcho: echo,
	CmdSet:  set,
	CmdGet:  get,
	CmdHSet: hset,
	CmdHGet: hget,
}

// ===== PING =====
func ping(args []Value) Value {
	reply := "PONG"
	if len(args) > 0 {
		reply = args[0].bulk
	}
	return Value{typ: RespTypeString, str: reply}
}

// ===== ECHO =====
func echo(args []Value) Value {
	if len(args) != 1 {
		return Value{typ: RespTypeError, str: "ERR wrong number of arguments for 'echo' command"}
	}
	return Value{typ: RespTypeBulk, bulk: args[0].bulk}
}

// ===== SET & GET =====
var SETs = map[string]string{}
var SETsMu = sync.RWMutex{}

func set(args []Value) Value {
	if len(args) != 2 {
		return Value{typ: RespTypeError, str: "ERR wrong number of arguments for 'set' command"}
	}

	key := args[0].bulk
	value := args[1].bulk

	// Write lock: Allow 1 writer, block readers and other writers
	SETsMu.Lock()
	SETs[key] = value
	SETsMu.Unlock()

	return Value{typ: RespTypeString, str: "OK"}
}

func get(args []Value) Value {
	if len(args) != 1 {
		return Value{typ: RespTypeError, str: "ERR wrong number of arguments for 'get' command"}
	}

	key := args[0].bulk

	// Read lock: Allow multiple readers, block writers
	SETsMu.RLock()
	value, ok := SETs[key]
	SETsMu.RUnlock()

	if !ok {
		return Value{typ: RespTypeNull}
	}

	return Value{typ: RespTypeBulk, bulk: value}
}

// ===== HSET & HGET =====
var HSETs = map[string]map[string]string{}
var HSETsMu = sync.RWMutex{}

func hset(args []Value) Value {
	if len(args) != 3 {
		return Value{typ: RespTypeError, str: "ERR wrong number of arguments for 'hset' command"}
	}

	hash := args[0].bulk
	key := args[1].bulk
	value := args[2].bulk

	HSETsMu.Lock()
	if _, ok := HSETs[hash]; !ok {
		HSETs[hash] = map[string]string{}
	}
	HSETs[hash][key] = value
	HSETsMu.Unlock()

	return Value{typ: RespTypeString, str: "OK"}
}

func hget(args []Value) Value {
	if len(args) != 2 {
		return Value{typ: RespTypeError, str: "ERR wrong number of arguments for 'hget' command"}
	}

	hash := args[0].bulk
	key := args[1].bulk

	HSETsMu.RLock()
	value, ok := HSETs[hash][key]
	HSETsMu.RUnlock()

	if !ok {
		return Value{typ: RespTypeNull}
	}

	return Value{typ: RespTypeBulk, bulk: value}
}
