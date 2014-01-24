package main

import "net"
import "fmt"

type Broadcast struct {
    Message []byte
    Sender  net.Conn
}

type ConnectionList []net.Conn

func broadcaster(conns []net.Conn, broadcastListener chan Broadcast) {
    for {
        b := <-broadcastListener
        for _, v := range conns {
            if v != nil && v != b.Sender {
                go v.Write(b.Message)
            }
        }
    }
}

func handler(conn net.Conn, broadcastListener chan Broadcast) {
    buf := make([]byte, 255)
    for {
        conn.Read(buf)
        broadcastListener <- Broadcast{buf, conn}
    }
}

func main() {
    conns := make([]net.Conn, 100)
    broadcastListener := make(chan Broadcast)
    go broadcaster(conns, broadcastListener)

    listener, err := net.Listen("tcp", ":8080")
    if err != nil {
        panic(err)
    }
    i := 0
    for {
        conn, err := listener.Accept()
        if err != nil {
            panic(err)
        }
        fmt.Println("Got connection from:", conn.RemoteAddr())
        conns[i] = conn
        i++
        go handler(conn, broadcastListener)
    }
}
