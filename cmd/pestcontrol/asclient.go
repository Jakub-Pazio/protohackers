package main

import "net"

type Client struct {
	conn net.Conn
}

func (c Client) sendMessage(msg Message) error {
	_, err := c.conn.Write(SerializeMessage(msg))
	return err
}

func (c Client) receiveHelloMessage() (HelloMessage, error) {
	return HelloMessage{}, nil
}
