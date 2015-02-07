package FNTP

import (
	"lib"
	"net"
)

const (
	PocketLength    = 100
	UdpPocketLength = PocketLength + 8
	IntMax          = 1<<31 - 1
)

var (
	StopBytes = []byte{'$', '$', '$', '$'}
)

type MetaData struct {
	DataId      uint32            //ID of Incoming Data identification
	Length      uint32            //Length of Incoming Data
	PocketCount uint32            //Incoming Data pockets count
	DataStack   map[uint32][]byte //index->buffer , dictionary of received bytes by indexes
	StopUDP     bool              //Stop Sending UDP data, used only for sending UDP data
}

type ErrorHandler func(error)
type DataReceive func([]byte)
type DisconnectedHandler func(error)
type UdpStopped func(MetaData)

type Socket struct {
	UdpSocket    *net.UDPConn        //Socket for Sending UDP data, for Server it's the same as Server Socket, for Client it's a new socket
	TCPSocket    *net.TCPConn        //TCP connection for client or server
	RemoteUDP    *net.UDPAddr        //Address for sending UDP data
	DataIn       map[uint32]MetaData //Dictionary for data incoming from This socket
	DataOut      map[uint32]MetaData //Dictionary for data outgoing from This socket
	Error        ErrorHandler        //Error handler function
	DataReceived DataReceive         // Data Receive Handler
	closed       bool
	Disconnected DisconnectedHandler
	IsServer     bool //Check is server side socket
	StoppedUDP   UdpStopped
}

//Crates MetaData from TCP received headers
func CreateMetaData(data []byte) (meta MetaData, converted bool) {
	converted = true
	meta.DataId, converted = lib.BytesToUint32(data[0:4])
	if !converted {
		lib.Log("Error Converting DataID !!")
		return
	}
	meta.Length, converted = lib.BytesToUint32(data[4:8])
	if !converted {
		lib.Log("Error Converting Length !!")
		return
	}
	meta.PocketCount = lib.CalcPocketCount(meta.Length, uint32(PocketLength))
	meta.StopUDP = false
	meta.DataStack = make(map[uint32][]byte)
	return
}

//Creates MetaData from Data to send
func CreateSendMetaData(data []byte) (meta MetaData, tcpData []byte) {
	meta.DataId = lib.Random(1, IntMax)
	tcpData = append(tcpData, lib.Uin32ToBytes(meta.DataId)...)
	meta.Length = uint32(len(data))
	tcpData = append(tcpData, lib.Uin32ToBytes(meta.Length)...)
	meta.PocketCount = lib.CalcPocketCount(meta.Length, uint32(PocketLength))
	meta.StopUDP = false
	meta.DataStack = make(map[uint32][]byte, int(meta.PocketCount))
	var (
		pos, next int
		current   uint32
	)
	data_id_byte := lib.Uin32ToBytes(meta.DataId)
	var append_data []byte
	for i := uint32(0); i < meta.PocketCount; i++ {
		current = uint32(i)
		pos = int(i * PocketLength)

		meta.DataStack[current] = append(meta.DataStack[current], data_id_byte...)
		meta.DataStack[current] = append(meta.DataStack[current], lib.Uin32ToBytes(current)...)
		next = pos + int(PocketLength)
		if uint32(next) > meta.Length {
			append_data = append(data[pos:], lib.FakeData((next - int(meta.Length)))...)
		} else {
			append_data = data[pos:next]
		}
		meta.DataStack[current] = append(meta.DataStack[current], append_data...)
	}
	return
}

func CreateSocket(tcp_connection *net.TCPConn, remote_udp *net.UDPAddr, udp_socket *net.UDPConn) (socket Socket, err error) {
	socket.TCPSocket = tcp_connection //this must be alive connection
	socket.TCPSocket.SetNoDelay(true)
	socket.RemoteUDP = remote_udp
	socket.UdpSocket = udp_socket
	socket.Error = func(e error) {
		lib.Log(e.Error())
	}
	socket.closed = false
	socket.DataReceived = func(data []byte) {}
	socket.DataIn = make(map[uint32]MetaData)
	socket.DataOut = make(map[uint32]MetaData)
	socket.Disconnected = func(err error) {}
	socket.StoppedUDP = func(m MetaData) {}
	return
}

//Reading Data from TCP connection, must be started concurrently and MAYBE WILL RUN IN EVENT LOOP
func (socket *Socket) ReadTCP() {
	buf_receive := make([]byte, 8) //If I'm typing 8 byte it's alwase giving EOF error from TCP
	for {
		if socket.closed {
			return
		}
		rlen, err := socket.TCPSocket.Read(buf_receive)
		if err != nil {
			socket.TCPSocket.Close()
			go socket.Disconnected(err)
			return
		}
		var buf []byte
		buf = append(buf, buf_receive[:rlen]...)
		if rlen != 8 {
			continue
		}

		data_id, converted := lib.BytesToUint32(buf[0:4])
		if !converted {
			lib.Log("Unable to convert data_id in TCP END DATA block")
			continue
		}
		if _, ok := socket.DataOut[data_id]; ok && string(buf[4:5]) == "$" {
			m := socket.DataOut[data_id]
			m.StopUDP = true
			socket.DataOut[data_id] = m
			continue
		}

		meta, converted := CreateMetaData(buf)
		if !converted {
			continue
		}
		socket.DataIn[meta.DataId] = meta
	}
}

//Sends TCP data to remote using Socket structure TCP connection
func (socket *Socket) WriteTCP(data []byte) (err error) {
	_, err = socket.TCPSocket.Write(data)
	if err != nil {
		socket.Error(err)
	}
	return
}

//Core function for sending data from socket, will contain TCP and UDP mixed send
func (socket *Socket) Send(data []byte) (err error) {
	meta, tcpData := CreateSendMetaData(data)
	err = socket.WriteTCP(tcpData)
	if err != nil {
		socket.Error(err)
		return
	}
	socket.DataOut[meta.DataId] = meta
	for {
		if socket.closed {
			return
		}
		if socket.DataOut[meta.DataId].StopUDP {
			go socket.StoppedUDP(socket.DataOut[meta.DataId])
			return
		}

		for _, v := range socket.DataOut[meta.DataId].DataStack {
			socket.WriteUDP(v)
		}
	}
}

//Read UDP data, right after UDP data recieve from UDP Socket
func (socket *Socket) ReadUDP(rlen int, buf_receive []byte, remote *net.UDPAddr) bool {
	if rlen != UdpPocketLength {
		return false
	}
	buf := buf_receive[:rlen]
	var (
		converted    bool
		data_id      uint32
		pocket_index uint32
	)
	data_id, converted = lib.BytesToUint32(buf[0:4])
	if !converted {
		lib.Log("Unable to convert data_id from UDP")
		return false
	}
	//If socket incoming data doesn't contains data_id return
	if _, ok := socket.DataIn[data_id]; !ok {
		return false
	}
	socket.RemoteUDP = remote //if DataID is same so maybe Remote address will change
	//If UDP data received for this data_id return
	if socket.DataIn[data_id].StopUDP {
		return true
	}

	pocket_index, converted = lib.BytesToUint32(buf[4:8])
	if !converted {
		lib.Log("Unable to convert pocket_index from UDP")
		return true
	}

	//If all data received
	if len(socket.DataIn[data_id].DataStack) == int(socket.DataIn[data_id].PocketCount) {
		m := socket.DataIn[data_id]
		m.StopUDP = true
		socket.DataIn[data_id] = m
		var stop_data []byte
		stop_data = append(stop_data, buf[0:4]...)
		stop_data = append(stop_data, StopBytes...)
		socket.WriteTCP(stop_data)
		all_data := lib.CombineBytesMap(socket.DataIn[data_id].DataStack)
		go socket.DataReceived(all_data[:socket.DataIn[data_id].Length])
		delete(socket.DataIn, data_id)
		return true
	}

	//if index already received return
	if _, ok := socket.DataIn[data_id].DataStack[pocket_index]; ok {
		return true
	}
	if socket.DataIn[data_id].DataStack == nil {
		m := socket.DataIn[data_id]
		m.DataStack = make(map[uint32][]byte)
		socket.DataIn[data_id] = m
	}
	socket.DataIn[data_id].DataStack[pocket_index] = append(socket.DataIn[data_id].DataStack[pocket_index], buf[8:]...)
	return true
}

//Send UDP data
func (socket *Socket) WriteUDP(data []byte) {
	var err error
	if socket.IsServer {
		_, err = socket.UdpSocket.WriteToUDP(data, socket.RemoteUDP)
	} else {
		_, err = socket.UdpSocket.Write(data)

	}
	if err != nil {
		socket.Error(err)
	}

}

func (socket *Socket) ListenUdp() {
	var buf = make([]byte, UdpPocketLength)
	for {
		if socket.closed {
			return
		}
		rlen, remote, receive_error := socket.UdpSocket.ReadFromUDP(buf)
		if receive_error != nil {
			socket.Error(receive_error)
			continue
		}
		//TODO: Think about making this as another chanel
		go socket.ReadUDP(rlen, buf[:rlen], remote)
	}
}

func (socket *Socket) Close() {
	socket.TCPSocket.Close()
	socket.UdpSocket.Close()
	socket.closed = true
	go socket.Disconnected(nil)
}
