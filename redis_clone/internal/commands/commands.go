package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ARtorias742/Redis/internal/store"
	"github.com/ARtorias742/Redis/internal/types"
)

func ExecuteCommand(s *store.Store, args []string, client *types.ClientState) string {
	if len(args) == 0 {
		return "-ERR empty command\r\n"
	}

	command := strings.ToUpper(args[0])
	switch command {
	case "SET":
		if len(args) < 3 {
			return "-ERR wrong number of arguments for 'set'\r\n"
		}
		s.SetString(args[1], args[2])
		return "+OK\r\n"
	case "GET":
		if len(args) != 2 {
			return "-ERR wrong number of arguments for 'get'\r\n"
		}
		if value, ok := s.GetString(args[1]); ok {
			return fmt.Sprintf("$%d\r\n%s\r\n", len(value), value)
		}
		return "$-1\r\n"
	case "DEL":
		if len(args) < 2 {
			return "-ERR wrong number of arguments for 'del'\r\n"
		}
		count := 0
		for _, key := range args[1:] {
			if s.Delete(key) {
				count++
			}
		}
		return fmt.Sprintf(":%d\r\n", count)
	case "EXPIRE":
		if len(args) != 3 {
			return "-ERR wrong number of arguments for 'expire'\r\n"
		}
		ttl, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			return "-ERR value is not an integer\r\n"
		}
		if value, ok := s.GetString(args[1]); ok {
			s.SetWithExpiry(args[1], value, ttl)
			return ":1\r\n"
		}
		return ":0\r\n"
	case "LPUSH":
		if len(args) < 3 {
			return "-ERR wrong number of arguments for 'lpush'\r\n"
		}
		s.ListPushLeft(args[1], args[2])
		return "+OK\r\n"
	case "RPOP":
		if len(args) != 2 {
			return "-ERR wrong number of arguments for 'rpop'\r\n"
		}
		if value, ok := s.ListPopRight(args[1]); ok {
			return fmt.Sprintf("$%d\r\n%s\r\n", len(value), value)
		}
		return "$-1\r\n"
	case "SADD":
		if len(args) < 3 {
			return "-ERR wrong number of arguments for 'sadd'\r\n"
		}
		s.SetAdd(args[1], args[2])
		return "+OK\r\n"
	case "SMEMBERS":
		if len(args) != 2 {
			return "-ERR wrong number of arguments for 'smembers'\r\n"
		}
		if members, ok := s.SetMembers(args[1]); ok {
			resp := fmt.Sprintf("*%d\r\n", len(members))
			for _, m := range members {
				resp += fmt.Sprintf("$%d\r\n%s\r\n", len(m), m)
			}
			return resp
		}
		return "*0\r\n"
	case "SYNC":
		// SYNC should be handled in server.go, not here
		// For now, return an error or placeholder since we can't access Server
		return "-ERR SYNC must be handled by the server\r\n"
	case "MULTI":
		if client.InTransaction {
			return "-ERR MULTI calls can not be nested\r\n"
		}
		client.InTransaction = true
		client.QueuedCommands = [][]string{}
		return "+OK\r\n"
	case "EXEC":
		if !client.InTransaction {
			return "-ERR EXEC without MULTI\r\n"
		}
		client.InTransaction = false
		results := s.ExecuteTransaction(client.QueuedCommands)
		client.QueuedCommands = nil
		resp := fmt.Sprintf("*%d\r\n", len(results))
		for _, r := range results {
			resp += r
		}
		return resp
	case "DISCARD":
		if !client.InTransaction {
			return "-ERR DISCARD without MULTI\r\n"
		}
		client.InTransaction = false
		client.QueuedCommands = nil
		return "+OK\r\n"
	default:
		return fmt.Sprintf("-ERR unknown command '%s'\r\n", command)
	}
}
