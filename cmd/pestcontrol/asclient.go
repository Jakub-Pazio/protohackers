package main

import (
	"bufio"
	"log"
	"net"
)

const (
	ASDomain = "pestcontrol.protohackers.com"
	ASPort   = "20547"
)

type Client struct {
	Site int
	conn net.Conn

	targets []TargetPopulation
}

func NewClient(site int) (Client, error) {
	asAddress := net.JoinHostPort(ASDomain, ASPort)
	conn, err := net.Dial("tcp", asAddress)

	if err != nil {
		return Client{}, err
	}

	client := Client{conn: conn, Site: site}
	if err = client.SendMessage(&ValidHelloMessage); err != nil {
		return client, err
	}

	if _, err = client.ReceiveHelloMessage(); err != nil {
		return client, err
	}

	msg, err := client.RecieveTargetPopulationMessage()
	log.Printf("Received Target from AS: %+v\n", msg)

	if err != nil {
		return client, nil
	}

	client.targets = msg.Targets

	return client, nil
}

func (c Client) SendMessage(msg Message) error {
	_, err := c.conn.Write(SerializeMessage(msg))
	return err
}

// TODO: think if adding timeout to the reading message, for example 5 sec, if no message we return error
func (c Client) ReceiveHelloMessage() (HelloMessage, error) {
	br := bufio.NewReader(c.conn)
	return ReadHelloMessage(br)
}

func (c Client) RecieveTargetPopulationMessage() (TargetPopulationMessage, error) {
	br := bufio.NewReader(c.conn)
	return ReadTargetPopulationsMessage(br)
}
