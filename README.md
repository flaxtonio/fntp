# About Protocol
FNTP is a combination of TCP and UDP as a mixed transport layer protocol. The basic thing is that TCP works slower than UDP but it's reliable and the basic idea of Flaxton FNTP protocol is to combine TCP reliable feature and UDP speed together. 
<img src="http://flaxton.io/img/fntp.png" alt="FNTP image"/>
Using this combination of TCP and UDP it makes possible to transfer all data faster using UDP and stay informed about sent data using reliable TCP.
<img src="http://flaxton.io/img/fntp1.png" alt="FNTP image"/>
# Protocol Implementation
This FNTP protocol implementation is written using Go programming language (<a href="http://golang.org" target="_blank">golang.org</a>).
Implementation in Node.js (<a href="http://nodejs.org" target="_blank">nodejs.org</a>) is available also here <a href="https://github.com/tigran-bayburtsyan/nodejs-flaxton" target="_blank">nodejs-flaxton</a>.
<br/>
<b>How it works</b>
<br/>
For example if you want to send 1 KB data from Client application to Server application, the FNTP workflow will look like this:
<ol>
<li>FNTP will create 8 byte header and will send it to Server application using TCP protocol. That 8 byte will contain 4 byte random generated integer as a unique ID for that 1 KB data, and 4 byte integer as a Length of sending data (in this example length will be 1000).</li>
<li>Server will receive that 8 byte and will calculate how many packages it will receive (UdpPackageCount) using Length parameter (from 8 byte header) and UDP Package Default length (from FNTP protocol constants).</li>
<li>Client will start sending UDP packages with position indexes until server wouldn't have all packages.</li>
<li>As UDP is not reliable, the most part of packages will be lost, but every package has its own index so if Server receives any package it will have that package position in all data combination.</li>
<li>After receiving all data packages from UDP (Server has to count saved in UdpPackageCount), Server will send 1 byte to Client (using TCP) telling the client that "all packages are received, stop sending UDP packages".</li>
</ol>

# Performance
As a result of using UDP for sending all data, FNTP is much faster than standard TCP, and because of some calculations and TCP based headers communications FNTP is a little bit slower than UDP, but FNTP is reliable as TCP.<br/>
With this combination FNTP becomes a new kind of transport layer protocol which allows to combine best features of TCP and UDP.

# "Hello World"
<b>Client Application</b>
```go
package main

import (
	"FNTP"
	"fmt"
)

func main() {
	client := FNTP.NewClient("127.0.0.1:8888")
	client.DataReceived = func(data []byte) {
		fmt.Println(string(data))
	}
	client.SendUdpStopped = func(m FNTP.MetaData) {
		fmt.Println(m)
	}
	client.Connect()
	client.Send([]byte("Sending Test String as first data"))
}
```
<b>Server Application</b>
```go
package main

import (
	"FNTP"
	"fmt"
)

func main() {
	server := FNTP.NewServer("127.0.0.1:8888")
	server.DataReceived = func(data []byte, socket *FNTP.Socket) {
		fmt.Println(string(data))
		socket.Send([]byte("Your Data Recieved. Thanks!"))
	}
	server.ErrorHandling = func(err error) {
		fmt.Println(err.Error())
	}
	server.Listen()
}
```
<b>Read <a href="https://github.com/flaxtonio/fntp/blob/master/tests/fntpTest.go" target="_blank"><code>tests/fntpTest.go</code></a> file for more detailed example</b>
