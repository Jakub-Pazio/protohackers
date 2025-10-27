package authority

import (
	"bufio"
	"context"
	"log"
	"net"

	"bean/cmd/pestcontrol/internal/animal"
	"bean/cmd/pestcontrol/internal/message"
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
	Site uint32
	conn net.Conn
	br   *bufio.Reader

	targets      []animal.TargetPopulation
	activePolicy map[string]PolicyStruct

	ActionChan chan func()
}

func (c *Client) Initialize() {
	for {
		log.Printf("Waiting for Site Adjustment for site: %d\n", c.Site)
		f := <-c.ActionChan
		log.Printf("Handling Site Adjustemnt for site: %d\n", c.Site)
		f()
	}
}

func NewClient(site uint32) (Client, error) {
	asAddress := net.JoinHostPort(ASDomain, ASPort)
	conn, err := net.Dial("tcp", asAddress)
	br := bufio.NewReader(conn)

	log.Printf("Created connection to AS: %q\n", asAddress)

	if err != nil {
		return Client{}, err
	}

	client := Client{
		conn:         conn,
		br:           br,
		Site:         site,
		activePolicy: make(map[string]PolicyStruct),
		ActionChan:   make(chan func()),
	}
	if err = client.SendMessage(&message.ValidHello); err != nil {
		return client, err
	}

	log.Printf("Sent Hello message to AS\n")

	if _, err = client.ReceiveHelloMessage(); err != nil {
		return client, err
	}

	log.Printf("Received Hello message fom AS\n")

	dialMsg := message.DialAuthority{Site: uint32(site)}
	if err = client.SendMessage(&dialMsg); err != nil {
		return client, err
	}

	msg, err := client.RecieveTargetPopulationMessage()
	log.Printf("Received Target from AS: %+v\n", msg)

	if err != nil {
		return client, err
	}

	client.targets = msg.Targets

	go client.Initialize()

	return client, nil
}

func (c Client) SendMessage(msg message.Message) error {
	_, err := c.conn.Write(message.Serialize(msg))
	return err
}

func (c *Client) AdjustPolicy(ctx context.Context, actual []message.Population) error {
	ch := make(chan error)
	//TODO: Get this tracer from the context
	// ctx, childSpan := tracer.Start(ctx, "adjust policy")
	// defer childSpan.End()

	log.Printf("Sending function to AP for population: %+v\n", actual)

	c.ActionChan <- func() {
		actMap := make(map[string]uint32)
		for _, a := range actual {
			actMap[a.Name] = a.Count
		}

		for _, target := range c.targets {
			specie := target.Specie
			actualCount := actMap[specie]

			if actualCount < target.Min || actualCount > target.Max {
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
					ch <- err
					return
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
						ch <- err
						return
					}
					delete(c.activePolicy, specie)
				}
			}
		}

		log.Printf("after adjusting: %d: %+v\n", c.Site, c.activePolicy)
		ch <- nil
	}
	return <-ch
}

func (c *Client) CreatePolicy(specie string, newPolicy Policy) (uint32, error) {
	msg := message.CreatePolicy{Specie: specie, Action: byte(newPolicy)}
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
	msg := message.DeletePolicy{PolicyID: currentPolicy.pid}
	if err := c.SendMessage(&msg); err != nil {
		return err
	}

	_, err := c.ReceiveOkMessage()
	return err
}

func (c Client) ReceivePolicyResultMessage() (message.PolicyResult, error) {
	return message.ReadPolicyResult(c.br)
}

func (c Client) ReceiveOkMessage() (message.OK, error) {
	return message.ReadOK(c.br)
}

// TODO: think if adding timeout to the reading message, for example 5 sec, if no message we return error
func (c Client) ReceiveHelloMessage() (message.Hello, error) {
	return message.ReadHello(c.br)
}

func (c Client) RecieveTargetPopulationMessage() (message.TargetPopulation, error) {
	return message.ReadTargetPopulations(c.br)
}
