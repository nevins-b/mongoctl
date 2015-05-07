package command

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/nevins-b/commgo"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type RemoveCommand struct {
	Meta
}

func (c *RemoveCommand) Run(args []string) int {
	var port int
	var ec2 bool
	var addr, username string
	flags := c.Meta.FlagSet("add", FlagSetDefault)
	flags.Usage = func() { c.Ui.Error(c.Help()) }
	flags.IntVar(&port, "port", 27017, "")
	flags.StringVar(&addr, "addr", "", "")
	flags.StringVar(&username, "username", "", "")
	flags.BoolVar(&ec2, "ec2", false, "")
	if err := flags.Parse(args); err != nil {
		return 1
	}

	if len(addr) == 0 && ec2 {
		resp, err := http.Get(ec2MetadataURI)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error: %s", err.Error()))
			return 1
		}

		out, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error: %s", err.Error()))
			return 1
		}

		addr = string(out)
		_, err = net.ResolveIPAddr("ip", addr)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error: %s", err.Error()))
			return 1
		}
	}

	node, err := c.Meta.GetNode()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error: %s", err.Error()))
		return 1
	}

	c.Ui.Info(fmt.Sprintf("Removing %s:%d from Cluster %s", addr, port, node))
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
	config.Version++

	host := fmt.Sprintf("%s:%d", addr, port)
	found := false
	for i, member := range config.Members {
		if member.Host == host {
			config.Members = append(config.Members[:i], config.Members[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		c.Ui.Error(fmt.Sprintf("Node %s not found in cluster", host))
		return 1
	}

	cmd := &bson.M{
		"replSetReconfig": config,
	}
	result := bson.M{}
	if err := session.DB("admin").Run(&cmd, &result); err != nil {
		c.Ui.Error(fmt.Sprintf("Error: %s", err.Error()))
		return 1
	}

	if c.Meta.consul {
		err = c.Meta.consulAgent.RemoveService(host)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error: %s", err.Error()))
			return 1
		}
	}
	return 0
}

func (c *RemoveCommand) Help() string {
	helpText := `
Usage: mongoctl remove [options]
  Remove a node from a Mongo Replica Set.
  This command connects to a Mongo server and removes the specified host from the cluster.
  If consul is specified this command will also remove
  the node from the Mongo service in consul.

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

  -ec2                    If the host to be added is an EC2 instance.
                          This can be used to discover the address of the
                          instance to add, assuming the command is run on the
													instance that is being added.
`
	return strings.TrimSpace(helpText)
}

func (c *RemoveCommand) Synopsis() string {
	return "Add a node to an existing mongo cluster"
}
