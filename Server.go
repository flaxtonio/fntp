package FNTP

import (
	"net"
)

type ServerDataReceive func([]byte, *Socket)
type NewClientHandler func(*Socket)

type Server struct {
	Address       string
	Sockets       []Socket
	udpConnection *net.UDPConn
	DataReceived  ServerDataReceive
	ErrorHandling ErrorHandler
	OnNewClient   NewClientHandler
}

func NewServer(address string) (server Server) {
	server.Address = address
	server.DataReceived = func(data []byte, socket *Socket) {}
	server.ErrorHandling = func(err error) {}
	server.OnNewClient = func(s *Socket) {}
	return
}

func (server *Server) Listen() {
	go server.udp_server()
	server.tcp_server()
}

func (s *Server) tcp_server() {
	addr, addr_err := net.ResolveTCPAddr("tcp", s.Address)
	if addr_err != nil {
		s.ErrorHandling(addr_err)
		return
	}

	socket, err := net.ListenTCP("tcp", addr)
	if err != nil {
		s.ErrorHandling(err)
		return
	}
	for {
		conn, packet_err := socket.AcceptTCP()
		if packet_err != nil {
			s.ErrorHandling(packet_err)
			continue
		}
		sc, err2 := CreateSocket(conn, nil, s.udpConnection)
		sc.IsServer = true
		if err2 != nil {
			continue
		}
		s.Sockets = append(s.Sockets, sc)
		index := len(s.Sockets) - 1
		s.Sockets[index].DataReceived = func(data []byte) {
			s.DataReceived(data, &s.Sockets[index])
		}
		s.Sockets[index].Disconnected = func(err error) {
			s.Sockets = append(s.Sockets[:index], s.Sockets[index+1:]...)
		}
		go s.OnNewClient(&s.Sockets[index])
		go s.Sockets[index].ReadTCP()
	}
}

func (s *Server) udp_server() {
	var (
		buf_receive = make([]byte, UdpPocketLength)
		err         error
	)
	addr, addr_err := net.ResolveUDPAddr("udp", s.Address)
	if addr_err != nil {
		s.ErrorHandling(addr_err)
		return
	}
	s.udpConnection, err = net.ListenUDP("udp", addr)
	if err != nil {
		s.ErrorHandling(err)
		return
	}

	for {
		rlen, remote, receive_err := s.udpConnection.ReadFromUDP(buf_receive)
		if receive_err != nil {
			s.ErrorHandling(receive_err)
			continue
		}
		if rlen != UdpPocketLength {
			continue
		}
		//TODO: Think about making this as another chanel
		//		go func() {
		for k, _ := range s.Sockets {
			go s.Sockets[k].ReadUDP(rlen, buf_receive[:rlen], remote)
		}
		//		}()
	}
}

func (server *Server) Disconnect() {
	for _, v := range server.Sockets {
		v.Close()
	}
}
