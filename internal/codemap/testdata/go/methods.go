package testfixture

// Server is a simple server type.
type Server struct {
	addr string
	port int
}

// Address returns the server address (value receiver).
func (s Server) Address() string { return s.addr }

// SetPort updates the server port (pointer receiver).
func (s *Server) SetPort(port int) { s.port = port }

// Send transmits a message and returns an error (pointer receiver, multiple params).
func (s *Server) Send(topic string, payload []byte) error { return nil }
