package command

import "strings"

type InitOrAddCommand struct {
	Meta
}

func (c *InitOrAddCommand) Run(args []string) int {

	nodes, err := c.Meta.consulAgent.GetService(
		c.Meta.consulKey,
		"",
	)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	if len(nodes) == 0 {
		c.Ui.Info("No nodes found in consul, running init")
		cmd := &InitCommand{
			Meta: c.Meta,
		}
		return cmd.Run(args)
	} else {
		c.Ui.Info("Cluster Found, adding node")
		cmd := &AddCommand{
			Meta: c.Meta,
		}
		return cmd.Run(args)
	}

}

func (c *InitOrAddCommand) Help() string {
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

func (c *InitOrAddCommand) Synopsis() string {
	return "Add a node to an existing mongo cluster"
}
