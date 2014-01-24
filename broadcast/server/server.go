package main

import (
    "fmt"
    "io"
    "net"
)

type Identifier string

type Message struct {
    Body      []byte
    Sender    Identifier
    Recipient Identifier
}

type Client struct {
    Conn       *net.Conn
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

func NewClient(conn *net.Conn) Client {
    return Client{Conn: conn, Messages: make(chan Message, 1024),
        Identifier: Identifier(fmt.Sprint((*conn).LocalAddr(), (*conn).RemoteAddr()))}
}

func MessageRouter(clients chan Client, messages chan Message) {
    clientList := make(map[Identifier]Client)
    for {
        select {
        case client := <-clients:
            clientList[client.Identifier] = client
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

func main() {
    clients := make(chan Client, 1024)
    messages := make(chan Message, 1024)
    go MessageRouter(clients, messages)

    server, err := net.Listen("tcp", ":8080")
    if err != nil {
        panic(err)
    }
    for {
        conn, err := server.Accept()
        if err != nil {
            panic(err)
        }
        client := NewClient(&conn)
        clients <- client
        go client.ReadHandler(messages)
        go client.WriteHandler()
    }
}
