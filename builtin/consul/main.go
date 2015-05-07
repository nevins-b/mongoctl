package consul

import (
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/consul/api"
)

type Agent struct {
	Server string
}

func (c *Agent) getClient() (cl *api.Client, err error) {
	config := api.DefaultConfig()

	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (c *Agent) GetAgent() (agent *api.Agent, err error) {
	client, err := c.getClient()
	if err != nil {
		return nil, err
	}
	return client.Agent(), nil
}

func (c *Agent) GetCatalog() (catalog *api.Catalog, err error) {
	client, err := c.getClient()
	if err != nil {
		return nil, err
	}
	return client.Catalog(), nil
}

func (c *Agent) AddService(addr, id, script, name string, port int) (err error) {
	agent, err := c.GetAgent()
	if err != nil {
		return err
	}
	check := api.AgentServiceCheck{
		Script:   fmt.Sprintf("/bin/nc -zv %s %d", addr, port),
		Interval: "15s",
	}
	service := &api.AgentServiceRegistration{
		Address: addr,
		Port:    port,
		Name:    "mongodb",
		ID:      fmt.Sprintf("%s:%d", addr, port),
		Check:   &check,
	}
	err = agent.ServiceRegister(service)
	if err != nil {
		return err
	}
	return nil
}

func (c *Agent) RemoveService(host string) (err error) {
	agent, err := c.GetAgent()
	if err != nil {
		return err
	}
	return agent.ServiceDeregister(host)
}

func (c *Agent) GetService(name, tag string) (nodes []*api.CatalogService, err error) {
	client, err := c.getClient()
	if err != nil {
		return nil, err
	}
	catalog := client.Catalog()
	options := &api.QueryOptions{
		WaitTime: 10 * time.Second,
	}
	nodes, _, err = catalog.Service(name, tag, options)
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, errors.New("No nodes found for service")
	}
	return nodes, nil
}
