package command

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nevins-b/commgo"
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
	defer session.Close()

	registered, err := c.Meta.consulAgent.GetService(c.Meta.consulKey, "")
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	status := &commgo.RsStatus{}
	cmd := &bson.M{
		"replSetGetStatus": "",
	}

	if err := session.DB("admin").Run(&cmd, &status); err != nil {
		c.Ui.Error(fmt.Sprintf("Error: %s", err.Error()))
		return 1
	}

	var live []*commgo.RsMemberStats
	var dead []*commgo.RsMemberStats
	for _, member := range status.Members {
		if member.State == 8 {
			dead = append(dead, member)
		} else {
			live = append(live, member)
		}
	}

	// Clean up nodes which are not live but are in consul
	for _, node := range registered {
		found := false
		for _, member := range live {
			parts := strings.Split(member.Name, ":")
			if len(parts) != 2 {
				c.Ui.Error(fmt.Sprintf("Can not parse node name %s", member.Name))
				continue
			}
			host := parts[0]
			port, _ := strconv.Atoi(parts[1])
			if node.Address == host && node.ServicePort == port {
				found = true
				break
			}
		}
		if !found {
			c.Ui.Info(fmt.Sprintf("Node %s not found, removing from Consul", node.ServiceID))
			err := c.Meta.consulAgent.RemoveService(node)
			if err != nil {
				c.Ui.Error(err.Error())
			}
		}
	}

	// Remove dead nodes from the Replica
	if len(dead) > 0 {
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

		config.Version++

		for _, member := range dead {
			c.Ui.Info(fmt.Sprintf("Removing dead host %s", member.Name))
			for i, host := range config.Members {
				if host.Host == member.Name {
					config.Members = append(config.Members[:i], config.Members[i+1:]...)
					break
				}
			}
		}

		cmd := &bson.M{
			"replSetReconfig": config,
		}
		result := bson.M{}
		if err := session.DB("admin").Run(&cmd, &result); err != nil {
			c.Ui.Error(fmt.Sprintf("Error: %s", err.Error()))
			return 1
		}

	}

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
	return "Remove dead nodes from Consul"
}
