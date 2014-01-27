package main

import (
    "fmt"
    "github.com/gorilla/mux"
    "github.com/gorilla/websocket"
    "io"
    "log"
    "net"
    "net/http"
)

type Identifier string

type Message struct {
    Body      []byte
    Sender    Identifier
    Recipient Identifier
}

type WSConn struct {
    *websocket.Conn
}

func (c WSConn) Read(p []byte) (n int, err error) {
    _, r, err := c.Conn.NextReader()
    if err != nil {
        return 0, err
    }
    n, err = r.Read(p)
    return n, err
}

func (c WSConn) Write(p []byte) (n int, err error) {
    w, err := c.Conn.NextWriter(1)
    defer func() {
        if r := recover(); r != nil {
            n = 0
            err = r.(error)
        }
    }()
    if err != nil {
        return 0, err
    }
    n, err = w.Write(p)
    w.Close()
    return n, err
}

type Conn interface {
    io.ReadWriter
    LocalAddr() net.Addr
    RemoteAddr() net.Addr
    Close() error
}

type Client struct {
    Conn       *Conn
    Messages   chan Message
    Identifier Identifier
    Closed     bool
}

func (client *Client) ReadHandler(messages chan Message) {
    conn := *client.Conn
    for {
        buf := make([]byte, 256)
        _, err := conn.Read(buf)
        if err != nil {
            if err != io.EOF {
                fmt.Printf("Read error in %s: %#v\n", client.Identifier, err)
                err = client.Close()
                if err != nil {
                    fmt.Printf("Close error in %s: %#v\n", client.Identifier, err)
                }
            }
            return
        }
        messages <- Message{buf, client.Identifier, "*"}
    }
}

func (client *Client) WriteHandler() {
    conn := *client.Conn
    for {
        message := <-client.Messages
        _, err := conn.Write(message.Body)
        if err != nil {
            if err != io.EOF {
                fmt.Printf("Write error in %s: %#v\n", client.Identifier, err)
                err = client.Close()
                if err != nil {
                    fmt.Printf("Close error in %s: %#v\n", client.Identifier, err)
                }
            }
            return
        }
    }
}

func (client *Client) Close() error {
    err := (*client.Conn).Close()
    client.Closed = true
    return err
}

func NewClient(conn *Conn) Client {
    return Client{Conn: conn, Messages: make(chan Message, 1024),
        Identifier: Identifier(fmt.Sprint((*conn).LocalAddr(), (*conn).RemoteAddr()))}
}

func MessageRouter(clients chan Client, messages chan Message) {
    clientList := make(map[Identifier]Client)
    for {
        select {
        case client := <-clients:
            clientList[client.Identifier] = client
            fmt.Println("Added:", client.Identifier)
        case message := <-messages:
            for _, client := range clientList {
                if client.Closed {
                    delete(clientList, client.Identifier)
                } else if message.Sender != client.Identifier {
                    client.Messages <- message
                }
            }
        }
    }

}

func socket(clients chan Client, messages chan Message) func(w http.ResponseWriter, r *http.Request) {
    return func(w http.ResponseWriter, r *http.Request) {
        conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
        if _, ok := err.(websocket.HandshakeError); ok {
            http.Error(w, "Not a websocket handshake", 400)
            return
        } else if err != nil {
            log.Println(err)
            return
        }

        ws := Conn(WSConn{conn})
        client := NewClient(&ws)
        clients <- client
        go client.ReadHandler(messages)
        go client.WriteHandler()
    }
}

func main() {
    clients := make(chan Client, 1024)
    messages := make(chan Message, 1024)
    go MessageRouter(clients, messages)

    router := mux.NewRouter()
    socketHandler := socket(clients, messages)
    router.HandleFunc("/socket", socketHandler)
    http.Handle("/", http.FileServer(http.Dir("./resources")))
    http.Handle("/socket", router)
    go http.ListenAndServe(":8081", nil)

    server, err := net.Listen("tcp", ":8080")
    if err != nil {
        panic(err)
    }
    for {
        conn, err := server.Accept()
        if err != nil {
            panic(err)
        }
        tcp := Conn(conn)
        client := NewClient(&tcp)
        clients <- client
        go client.ReadHandler(messages)
        go client.WriteHandler()
    }
}
