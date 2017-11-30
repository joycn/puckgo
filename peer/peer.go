package peer

// Peer is the inerface wraps the Read, Write method.
type Peer interface {
	Read(p []byte) (n int, err error)
	Write(p []byte) (n int, err error)
}
