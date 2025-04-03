package types

// ClientState holds transaction-related state for a client connection.
type ClientState struct {
	InTransaction  bool
	QueuedCommands [][]string
}
