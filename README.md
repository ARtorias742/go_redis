A Redis-like in-memory store with persistence, replication, and advanced data structures.

## Features
- **Key-Value**: SET, GET, DEL, EXPIRE
- **Lists**: LPUSH, RPOP
- **Sets**: SADD, SMEMBERS
- **Persistence**: RDB snapshots and AOF logging
- **Replication**: Master-replica sync (basic)
- **Logging**: File-based logs
- **Configuration**: YAML file

## Running the Project
1. Initialize the project:
   ```bash
   go mod init github.com/yourusername/redis-clone