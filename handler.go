package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

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

// ==== Helpers =====

func checkArgsCount(command string, args []Value, expected int) *Value {
	if len(args) == expected {
		return nil
	}

	return &Value{
		typ: RespTypeError,
		str: fmt.Sprintf("ERR wrong number of arguments for '%s' command", command),
	}
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
	err_val := checkArgsCount(CmdEcho, args, 1)
	if err_val != nil {
		return *err_val
	}
	return Value{typ: RespTypeBulk, bulk: args[0].bulk}
}

// ===== SET & GET =====

var SETs = map[string]string{}
var SETsMu = sync.RWMutex{}

// Note:
// - Write lock: Allow 1 writer, block readers and other writers
// - Read lock: Allow multiple readers, block writers

// Map SET key -> Unix timestamp in ms
var SETsExpirations = map[string]int64{}
var SETsExpirationsMu = sync.RWMutex{}

func set(args []Value) Value {
	if len(args) < 2 {
		return Value{
			typ: RespTypeError,
			str: "ERR wrong number of arguments for 'SET' command",
		}
	}

	key := args[0].bulk
	value := args[1].bulk

	// Default: no expiration
	var expiresAtMs int64 = 0

	// Parse optional arguments
	for i := 2; i < len(args); i++ {
		option := strings.ToUpper(args[i].bulk)

		switch option {
		case "PX":
			// Handle PX milliseconds ()
			if i+1 > len(args) {
				return Value{
					typ: RespTypeError,
					str: "ERR SET PX requires an argument",
				}
			}

			ms, err := strconv.ParseInt(args[i+1].bulk, 10, 64)
			if err != nil || ms <= 0 {
				return Value{
					typ: RespTypeError,
					str: "ERR SET PX value must be a positive integer",
				}
			}

			expiresAtMs = time.Now().UnixMilli() + ms
			i++ // skip the value argument
		default:
			return Value{
				typ: RespTypeError,
				str: "ERR unknown option '%s' for 'SET' command",
			}
		}
	}

	SETsMu.Lock()
	SETs[key] = value
	SETsMu.Unlock()

	// Set expiration if specified
	if expiresAtMs > 0 {
		SETsExpirationsMu.Lock()
		SETsExpirations[key] = expiresAtMs
		SETsExpirationsMu.Unlock()
	}

	return Value{typ: RespTypeString, str: "OK"}
}

func get(args []Value) Value {
	err_val := checkArgsCount(CmdGet, args, 1)
	if err_val != nil {
		return *err_val
	}

	key := args[0].bulk

	// Check if the key has expired
	SETsExpirationsMu.RLock()
	expiresAtMs, ok := SETsExpirations[key]
	SETsExpirationsMu.RUnlock()

	// Handle expired key: delete from both maps and return null
	if ok && time.Now().UnixMilli() > expiresAtMs {
		SETsMu.Lock()
		delete(SETs, key)
		SETsMu.Unlock()

		SETsExpirationsMu.Lock()
		delete(SETsExpirations, key)
		SETsExpirationsMu.Unlock()

		return Value{typ: RespTypeNull}
	}

	// Read value
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
	err_val := checkArgsCount(CmdHSet, args, 3)
	if err_val != nil {
		return *err_val
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
	err_val := checkArgsCount(CmdHGet, args, 2)
	if err_val != nil {
		return *err_val
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
