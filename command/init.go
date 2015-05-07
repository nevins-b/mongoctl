package command

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type InitCommand struct {
	Meta
}

func (c *InitCommand) Run(args []string) int {
	var username string
	flags := c.Meta.FlagSet("init", FlagSetDefault)
	flags.Usage = func() { c.Ui.Error(c.Help()) }
	flags.StringVar(&username, "username", "", "")

	if err := flags.Parse(args); err != nil {
		return 1
	}

	node, err := c.Meta.GetNode()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error: %s", err.Error()))
		return 1
	}
	info := &mgo.DialInfo{
		Addrs:    []string{node},
		Timeout:  5 * time.Second,
		Username: username,
		Direct:   true,
	}

	if len(username) > 0 {
		info.Password, _ = c.Ui.Ask("Password: ")
	}
	session, err := mgo.DialWithInfo(info)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// This needs to be set since we are working
	// with a single node not a cluster yet
	session.SetMode(mgo.Monotonic, true)

	defer session.Close()
	cmd := &bson.M{
		"replSetInitiate": ""}
	result := bson.M{}

	if err := session.DB("admin").Run(&cmd, &result); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	if c.Meta.consul {
		addr, err := c.Meta.GetLocalIP()
		port := 27017
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error: %s", err.Error()))
			return 1
		}
		err = c.Meta.consulAgent.AddService(
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

	out := fmt.Sprintf("%v", result)
	c.Ui.Output(out)
	return 0
}

func (c *InitCommand) Help() string {
	helpText := `
Usage: mongoctl init [options]
  Initialize a new Mongo Replica Set.
  This command connects to a Mongo server and initilizes a cluster.

General Options:
  -mongo=addr             The address of the Mongo server if not using Consul.

  -consul-service=service The service name to use when looking up Mongo
	                        with consul.

	-consul-server=addr			The address of the consul server to use,
	                        this defaults to 127.0.0.1:8500.
  -consul                 Use consul to find Mongo

Init Options:

	-username=username      The username to authenticate with if required.

`
	return strings.TrimSpace(helpText)
}

func (c *InitCommand) Synopsis() string {
	return "Initilize a new replica set"
}
