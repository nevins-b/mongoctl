package command

import (
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/nevins-b/commgo"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type AddCommand struct {
	Meta
}

func (c *AddCommand) Run(args []string) int {
	var priority, port int
	var hidden, arbitrator bool
	var addr, username string
	flags := c.Meta.FlagSet("add", FlagSetDefault)
	flags.Usage = func() { c.Ui.Error(c.Help()) }
	flags.IntVar(&priority, "priority", 1, "")
	flags.IntVar(&port, "port", 27017, "")
	flags.StringVar(&addr, "addr", "", "")
	flags.StringVar(&username, "username", "", "")
	flags.BoolVar(&hidden, "hidden", false, "")
	flags.BoolVar(&arbitrator, "arbitrator", false, "")
	if err := flags.Parse(args); err != nil {
		return 1
	}

	node, err := c.Meta.GetNode()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	info := &mgo.DialInfo{
		Addrs:    []string{node},
		Timeout:  5 * time.Second,
		Username: username,
	}

	if len(username) > 0 {
		info.Password, _ = c.Ui.Ask("Password: ")
	}
	session, err := mgo.DialWithInfo(info)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	defer session.Close()

	config := &commgo.RsConf{}
	conn := session.DB("local").C("system.replset")
	count, err := conn.Count()
	if count > 1 {
		c.Ui.Error("error: local.system.replset has unexpected contents")
		return 1
	}
	err = conn.Find(bson.M{}).One(&config)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	config.Version++

	var max int64
	max = 0
	for _, member := range config.Members {
		if member.ID > max {
			max = member.ID
		}
	}
	cfg := &commgo.Host{
		ID:          max + 1,
		Host:        fmt.Sprintf("%s:%d", addr, port),
		ArbiterOnly: arbitrator,
	}

	config.Members = append(config.Members, cfg)

	cmd := &bson.M{
		"replSetReconfig": config,
	}
	result := bson.M{}
	if err := session.DB("admin").Run(&cmd, &result); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	if c.Meta.consul {
		agent, err := c.Meta.GetConsulAgent()
		if err != nil {
			c.Ui.Error(err.Error())
			return 1
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
			c.Ui.Error(err.Error())
			return 1
		}
	}
	return 0
}

func (c *AddCommand) Help() string {
	helpText := `
Usage: mongoctl add [options]
  Add a node to an existing Mongo Replica Set.
  This command connects to a Mongo server and adds the specified host to an
	existing cluster. If consul is specified this command will also register
	the added host to the Mongo service in consul.

General Options:
  -mongo=addr             The address of the Mongo server if not using Consul.

  -consul-service=service The service name to use when looking up Mongo
	                        with consul.

	-consul-server=addr			The address of the consul server to use,
	                        this defaults to 127.0.0.1:8500.
  -consul                 Use consul to find Mongo

Init Options:

	-username=username      The username to authenticate with if required.

	-addr=addr							The address of the host to add.

	-port=port							The port of the host to add.
													Defaults to 27017.

	-priority=priority			The priority of the host to add.
													Defaults to 1.

	-hidden									If the host should be added hidden.
													Defaults to False.

	-arbitrator							If the host should be added as an arbitrator.
													Defaults to False.
`
	return strings.TrimSpace(helpText)
}

func (c *AddCommand) Synopsis() string {
	return "Add a node to an existing mongo cluster"
}
