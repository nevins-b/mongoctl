package command

import (
	"fmt"
	"strings"
	"time"

	"github.com/nevins-b/commgo"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type AddCommand struct {
	Meta
}

func (c *AddCommand) Run(args []string) int {
	var priority, port int
	var hidden, arbitrator, ec2 bool
	var addr, username string
	flags := c.Meta.FlagSet("add", FlagSetDefault)
	flags.Usage = func() { c.Ui.Error(c.Help()) }
	flags.IntVar(&priority, "priority", 1, "")
	flags.IntVar(&port, "port", 27017, "")
	flags.StringVar(&addr, "addr", "", "")
	flags.StringVar(&username, "username", "", "")
	flags.BoolVar(&hidden, "hidden", false, "")
	flags.BoolVar(&arbitrator, "arbitrator", false, "")
	flags.BoolVar(&ec2, "ec2", false, "")
	if err := flags.Parse(args); err != nil {
		return 1
	}

	if len(addr) == 0 && ec2 {
		ip, err := c.Meta.GetLocalIP()
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error: %s", err.Error()))
			return 1
		}
		addr = ip
	}

	node, err := c.Meta.GetNode()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error: %s", err.Error()))
		return 1
	}

	c.Ui.Info(fmt.Sprintf("Adding %s:%d to Cluster %s", addr, port, node))
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
		c.Ui.Error(fmt.Sprintf("Error: %s", err.Error()))
		return 1
	}

	defer session.Close()

	config := &commgo.RsConf{}
	conn := session.DB("local").C("system.replset")
	count, err := conn.Count()
	if count > 1 {
		c.Ui.Error("Error: local.system.replset has unexpected contents")
		return 1
	}
	err = conn.Find(bson.M{}).One(&config)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error: %s", err.Error()))
		return 1
	}

	var max int64
	max = 0
	host := fmt.Sprintf("%s:%d", addr, port)
	exists := false
	for _, member := range config.Members {
		if member.ID > max {
			max = member.ID
			if member.Host == host {
				exists = true
				break
			}
		}
	}

	if !exists {
		config.Version++

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
			c.Ui.Error(fmt.Sprintf("Error: %s", err.Error()))
			return 1
		}
	}

	if c.Meta.consul {
		registered, err := c.Meta.consulAgent.GetService(c.Meta.consulKey, "")
		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}
		found := false
		for _, node := range registered {
			if node.Address == host && node.ServicePort == port {
				found = true
				break
			}
		}
		if !found {
			err := c.Meta.consulAgent.AddService(
				addr,
				fmt.Sprintf("%s:%d", addr, port),
				fmt.Sprintf("/bin/nc -zv %s %d", addr, port),
				"mongodb",
				port,
			)
			if err != nil {
				c.Ui.Error(fmt.Sprintf("Error: %s", err.Error()))
				return 1
			}
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

  -consul-server=addr     The address of the consul server to use,
                          this defaults to 127.0.0.1:8500.
  -consul                 Use consul to find Mongo

Init Options:

  -username=username      The username to authenticate with if required.

  -addr=addr              The address of the host to add.

  -port=port              The port of the host to add.
                          Defaults to 27017.

  -priority=priority      The priority of the host to add.
                          Defaults to 1.

  -hidden                 If the host should be added hidden.
                          Defaults to False.

  -arbitrator             If the host should be added as an arbitrator.
                          Defaults to False.

  -ec2                    If the host to be added is an EC2 instance.
                          This can be used to discover the address of the
                          instance to add, assuming the command is run on the
													instance that is being added.
`
	return strings.TrimSpace(helpText)
}

func (c *AddCommand) Synopsis() string {
	return "Add a node to an existing mongo cluster"
}
