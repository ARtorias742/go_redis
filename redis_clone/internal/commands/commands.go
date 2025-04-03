package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ARtorias742/redis_clone/internal/store"
)

func ExecuteCommand(s *store.Store, args []string) string {
	if len(args) == 0 {
		return "-ERR empty command\r\n"
	}

	command := strings.ToUpper(args[0])
	switch command {
	case "SET":
		if len(args) < 3 {
			return "-ERR wrong number of arguments for 'set' command\r\n"
		}
		s.Set(args[1], args[2])
		return "+OK\r\n"

	case "GET":
		if len(args) != 2 {
			return "-ERR wrong number of arguments for 'set' command\r\n"
		}
		if value, ok := s.Get(args[1]); ok {
			return fmt.Sprintf("$%d\r\n%s\r\n", len(value), value)
		}

		return "$-1\r\n"

	case "DEL":
		if len(args) < 2 {
			return "-ERR wrong number of arguments for 'del' command\r\n"
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
			return "-ERR wrong number of arguments for 'expire' command\r\n"
		}
		ttl, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			return "-ERR value is not an integer or out of range\r\n"
		}

		if value, ok := s.Get(args[1]); ok {
			s.SetWithExpiry(args[1], value, ttl)
			return ":1\r\n"
		}
		return ":0\r\n"

	case "EXISTS":
		if len(args) != 2 {
			return "-ERR wrong number of arguments for 'exists' command\r\n"
		}
		if s.Exists(args[1]) {
			return ":1\r\n"
		}
		return ":0\r\n"

	default:
		return fmt.Sprintf("-ERR unknown command '%s'\r\n", command)
	}
}
