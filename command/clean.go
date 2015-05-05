package command

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type CleanCommand struct {
	Meta
}

func (c *CleanCommand) Run(args []string) int {
	var username string
	flags := c.Meta.FlagSet("clean", FlagSetDefault)
	flags.Usage = func() { c.Ui.Error(c.Help()) }
	flags.StringVar(&username, "username", "", "")

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

	nodeList := session.LiveServers()
	catalog, err := c.Meta.GetConsulCatalog()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	registered, _, err := catalog.Service(c.Meta.consulKey, "", nil)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	agent, err := c.Meta.GetConsulAgent()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	for _, node := range registered {
		found := false
		for _, r := range nodeList {
			parts := strings.Split(r, ":")
			if len(parts) != 2 {
				c.Ui.Error(fmt.Sprintf("Can not parse node %s", r))
				continue
			}
			host := parts[0]
			port, _ := strconv.Atoi(parts[1])
			if node.ServiceAddress == host && node.ServicePort == port {
				found = true
				break
			}
		}
		if !found {
			err := agent.ServiceDeregister(node.ServiceID)
			if err != nil {
				c.Ui.Error(err.Error())
			}
		}
	}

	defer session.Close()
	cmd := &bson.M{
		"replSetInitiate": ""}
	result := bson.M{}

	if err := session.DB("admin").Run(&cmd, &result); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	out := fmt.Sprintf("%v", result)
	c.Ui.Output(out)
	return 0
}

func (c *CleanCommand) Help() string {
	helpText := `
Usage: mongoctl clean [options]
  Get the status of a Mongo Cluster
  This command connects to a Mongo server and retrieves the status
	of the cluster.

General Options:
  -mongo=addr             The address of the Mongo server if not using Consul.

  -consul-service=service The service name to use when looking up Mongo
	                        with consul.

	-consul-server=addr			The address of the consul server to use,
	                        this defaults to 127.0.0.1:8500.
  -consul                 Use consul to find Mongo

Clean Options:

	-username=username      The username to authenticate with if required.
`
	return strings.TrimSpace(helpText)
}

func (c *CleanCommand) Synopsis() string {
	return "Initilize a new replica set"
}
