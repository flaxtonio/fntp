package main

import (
	"FNTP"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) >= 3 {
		switch os.Args[1] {
		case "client":
			{
				client := FNTP.NewClient(os.Args[2])
				client.DataReceived = func(data []byte) {
					fmt.Println(string(data))
				}
				client.SendUdpStopped = func(m FNTP.MetaData) {
					fmt.Println(m)
				}
				client.Connect()
				client.Send([]byte("Bbbbbbbbbb"))
				for {

				}
			}
		case "server":
			{
				server := FNTP.NewServer(os.Args[2])
				server.DataReceived = func(data []byte, socket *FNTP.Socket) {
					fmt.Println(string(data))
					socket.Send([]byte("aaaaaaaaaaaaaa"))
				}
				server.ErrorHandling = func(err error) {
					fmt.Println("vvvvvvvv", err.Error())
				}
				server.Listen()
			}
		}
	}
}
