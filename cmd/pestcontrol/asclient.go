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

type Policy byte

type PolicyStruct struct {
	pid    uint32
	policy Policy
}

const (
	Cull     Policy = 0x90
	Conserve Policy = 0xa0
)

type Client struct {
	Site int
	conn net.Conn
	br   *bufio.Reader

	targets      []TargetPopulation
	activePolicy map[string]PolicyStruct
}

func NewClient(site int) (Client, error) {
	asAddress := net.JoinHostPort(ASDomain, ASPort)
	conn, err := net.Dial("tcp", asAddress)
	br := bufio.NewReader(conn)

	log.Printf("Created connection to AS: %q\n", asAddress)

	if err != nil {
		return Client{}, err
	}

	client := Client{conn: conn, br: br, Site: site, activePolicy: make(map[string]PolicyStruct)}
	if err = client.SendMessage(&ValidHelloMessage); err != nil {
		return client, err
	}

	log.Printf("Sent Hello message to AS\n")

	if _, err = client.ReceiveHelloMessage(); err != nil {
		return client, err
	}

	log.Printf("Received Hello message fom AS\n")

	dialMsg := DialAuthorityMessage{Site: uint32(site)}
	if err = client.SendMessage(&dialMsg); err != nil {
		return client, err
	}

	msg, err := client.RecieveTargetPopulationMessage()
	log.Printf("Received Target from AS: %+v\n", msg)

	if err != nil {
		return client, err
	}

	client.targets = msg.Targets

	return client, nil
}

func (c Client) SendMessage(msg Message) error {
	_, err := c.conn.Write(SerializeMessage(msg))
	return err
}

func (c *Client) AdjustPolicy(actual []Population) error {
	actMap := make(map[string]uint32)
	for _, a := range actual {
		actMap[a.Name] = a.Count
	}

	for _, target := range c.targets {
		specie := target.Specie
		actualCount := actMap[specie]

		if actualCount < target.Min || actualCount > target.Max {
			//TODO: check if we have correct policy, if not remove or/and add new
			currentPolicy, ok := c.activePolicy[specie]
			if ok {
				if actualCount < target.Min && currentPolicy.policy == Conserve {
					continue
				}
				if actualCount > target.Max && currentPolicy.policy == Cull {
					continue
				}
				c.CancelPolicy(currentPolicy)
				delete(c.activePolicy, specie)
			}

			var newPolicy Policy
			if actualCount < target.Min {
				newPolicy = Conserve
			} else {
				newPolicy = Cull
			}

			id, err := c.CreatePolicy(specie, newPolicy)
			if err != nil {
				//TODO: if we treat it as a transaction we should remove previous
				// added policies
				return err
			}
			c.activePolicy[specie] = PolicyStruct{
				pid:    id,
				policy: newPolicy,
			}

		} else {
			// current number of animals is correct, remove policy if exists
			if currentPolicy, ok := c.activePolicy[specie]; ok {
				err := c.CancelPolicy(currentPolicy)
				if err != nil {
					return err
				}
				delete(c.activePolicy, specie)
			}
		}
	}

	return nil
}

func (c *Client) CreatePolicy(specie string, newPolicy Policy) (uint32, error) {
	msg := CreatePolicyMessage{Specie: specie, Action: byte(newPolicy)}
	if err := c.SendMessage(&msg); err != nil {
		return 0, err
	}

	resultMsg, err := c.ReceivePolicyResultMessage()
	if err != nil {
		return 0, err
	}

	return resultMsg.PolicyID, nil
}

func (c *Client) CancelPolicy(currentPolicy PolicyStruct) error {
	msg := DeletePolicyMessage{PolicyID: currentPolicy.pid}
	return c.SendMessage(&msg)
}

func (c Client) ReceivePolicyResultMessage() (PolicyResultMessage, error) {
	return ReadPolicyResultMessage(c.br)
}

// TODO: think if adding timeout to the reading message, for example 5 sec, if no message we return error
func (c Client) ReceiveHelloMessage() (HelloMessage, error) {
	return ReadHelloMessage(c.br)
}

func (c Client) RecieveTargetPopulationMessage() (TargetPopulationMessage, error) {
	return ReadTargetPopulationsMessage(c.br)
}
