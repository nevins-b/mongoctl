package command

import (
	"fmt"
	"strings"
	"time"

	"github.com/nevins-b/commgo"
	"gopkg.in/mgo.v2"
)

type StatusCommand struct {
	Meta
}

func (c *StatusCommand) Run(args []string) int {
	var username string
	flags := c.Meta.FlagSet("init", FlagSetDefault)
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

	defer session.Close()
	result := &commgo.RsStatus{}

	if err := session.DB("admin").Run("replSetGetStatus", result); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	c.Ui.Output("Node\t\tState\t\tLast Heartbeat")
	for _, member := range result.Members {
		var out string
		if member.LastHeartbeat != nil {
			out = fmt.Sprintf("%s\t\t%s\t\t%v", member.Name, member.StateStr, member.LastHeartbeat)
		} else {
			out = fmt.Sprintf("%s\t\t%s", member.Name, member.StateStr)
		}

		c.Ui.Output(out)
	}
	return 0
}

func (c *StatusCommand) Help() string {
	helpText := `
Usage: mongoctl status [options]
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

Status Options:

	-username=username      The username to authenticate with if required.

`
	return strings.TrimSpace(helpText)
}

func (c *StatusCommand) Synopsis() string {
	return "Get the status of a Mongo Cluster"
}
