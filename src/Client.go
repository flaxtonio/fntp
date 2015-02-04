package FNTP

import (
	"net"
)

type Client struct {
	Address        string
	Socket         Socket
	DataReceived   DataReceive
	ErrorHandling  ErrorHandler
	Disconnected   DisconnectedHandler
	SendUdpStopped UdpStopped
}

func NewClient(address string) (client Client) {
	client.Address = address
	client.DataReceived = func(data []byte) {}
	client.ErrorHandling = func(err error) {}
	client.Disconnected = func(err error) {}
	client.SendUdpStopped = func(m MetaData) {}
	return
}

//Makes TCP connection and adds Socket
func (client *Client) Connect() (err error) {
	err = nil
	var (
		tcpAddr *net.TCPAddr
		tcpConn *net.TCPConn
		udpAddr *net.UDPAddr
		udpConn *net.UDPConn
	)
	tcpAddr, err = net.ResolveTCPAddr("tcp", client.Address)
	if err != nil {
		return
	}
	udpAddr, err = net.ResolveUDPAddr("udp", client.Address)
	if err != nil {
		return
	}
	tcpConn, err = net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return
	}
	udpConn, err = net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return
	}
	client.Socket, err = CreateSocket(tcpConn, udpAddr, udpConn)
	if err != nil {
		return
	}
	client.Socket.IsServer = false
	client.Socket.DataReceived = client.DataReceived
	client.Socket.Error = client.ErrorHandling
	client.Socket.StoppedUDP = client.SendUdpStopped

	//Start Receiving Threads
	go client.Socket.ReadTCP()
	go client.Socket.ListenUdp()

	return
}

func (client *Client) Send(data []byte) {
	client.Socket.Send(data)
}

func (client *Client) Disconnect() {
	client.Socket.Close()
}
