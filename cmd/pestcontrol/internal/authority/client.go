package authority

import (
	"context"
	"fmt"

	"bean/cmd/pestcontrol/internal/animal"
	"bean/cmd/pestcontrol/internal/message"
	"bean/cmd/pestcontrol/internal/pcnet"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
)

const (
	ASDomain = "pestcontrol.protohackers.com"
	ASPort   = "20547"
)

const name = "jakubpazio.site/protohackers/authority/client"

var (
	logger = otelslog.NewLogger(name)
	tracer = otel.Tracer(name)
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
	conn *pcnet.Conn

	targets      []animal.TargetPopulation
	activePolicy map[string]PolicyStruct

	ActionChan chan func()
}

func (c *Client) Initialize(ctx context.Context) {
	for {
		f := <-c.ActionChan
		f()
	}
}

func NewClient(ctx context.Context, site uint32, conn *pcnet.Conn) (*Client, error) {
	logger.InfoContext(ctx, "Created connecton to AS", "site", site)

	client := &Client{
		conn:         conn,
		Site:         site,
		activePolicy: make(map[string]PolicyStruct),
		ActionChan:   make(chan func()),
	}
	go client.Initialize(ctx)

	if err := conn.Write(ctx, message.ValidHello); err != nil {
		return client, fmt.Errorf("write Hello: %w", err)
	}
	logger.InfoContext(ctx, "Send Hello to AS", "site", site)

	if _, err := conn.ReadHello(ctx); err != nil {
		return client, fmt.Errorf("read Hello: %w", err)
	}
	logger.InfoContext(ctx, "Received Hello", "site", site)

	dialMsg := message.DialAuthority{Site: uint32(site)}
	if err := conn.Write(ctx, &dialMsg); err != nil {
		return client, fmt.Errorf("write DialAutority: %w", err)
	}

	msg, err := client.conn.ReadTargetPopulation(ctx)
	if err != nil {
		return client, fmt.Errorf("receive target population: %w", err)
	}
	logger.InfoContext(ctx, "Received target from AS", "site", site, "target length", len(msg.Targets))

	client.targets = msg.Targets

	return client, nil
}

func (c *Client) AdjustPolicy(ctx context.Context, current []message.Population) error {
	ch := make(chan error)
	ctx, span := tracer.Start(ctx, "adjust-policy")
	defer span.End()

	logger.InfoContext(ctx, "Adjusting policy", "site", c.Site)

	c.ActionChan <- func() {
		actMap := make(map[string]uint32)
		for _, a := range current {
			actMap[a.Name] = a.Count
		}

		for _, target := range c.targets {
			specie := target.Specie
			actualCount := actMap[specie]

			if actualCount < target.Min || actualCount > target.Max {
				logger.InfoContext(ctx, "Population out of range", "specie", target.Specie)
				currentPolicy, ok := c.activePolicy[specie]
				if ok {
					if actualCount < target.Min && currentPolicy.policy == Conserve {
						continue
					}
					if actualCount > target.Max && currentPolicy.policy == Cull {
						continue
					}
					c.CancelPolicy(ctx, currentPolicy)
					delete(c.activePolicy, specie)
				}

				newPolicy := Cull
				if actualCount < target.Min {
					newPolicy = Conserve
				}

				id, err := c.CreatePolicy(ctx, specie, newPolicy)
				if err != nil {
					ch <- fmt.Errorf("create policy: %w", err)
					return
				}
				c.activePolicy[specie] = PolicyStruct{
					pid:    id,
					policy: newPolicy,
				}

			} else {
				// current number of animals is correct, remove policy if exists
				if currentPolicy, ok := c.activePolicy[specie]; ok {
					err := c.CancelPolicy(ctx, currentPolicy)
					if err != nil {
						ch <- fmt.Errorf("cancel policy: %w", err)
						return
					}
					delete(c.activePolicy, specie)
				}
			}
		}
		ch <- nil
	}
	return <-ch
}

func (c *Client) CreatePolicy(ctx context.Context, specie string, newPolicy Policy) (uint32, error) {
	msg := message.CreatePolicy{Specie: specie, Action: byte(newPolicy)}
	if err := c.conn.Write(ctx, &msg); err != nil {
		return 0, fmt.Errorf("send create policy message: %w", err)
	}

	resultMsg, err := c.conn.ReadPolicyResult(ctx)
	if err != nil {
		return 0, fmt.Errorf("receive policy result: %w", err)
	}

	return resultMsg.PolicyID, nil
}

func (c *Client) CancelPolicy(ctx context.Context, currentPolicy PolicyStruct) error {
	msg := message.DeletePolicy{PolicyID: currentPolicy.pid}
	if err := c.conn.Write(ctx, &msg); err != nil {
		return fmt.Errorf("delete policy: %w", err)
	}

	_, err := c.conn.ReadOK(ctx)
	return err
}
