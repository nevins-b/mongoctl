package command

import (
	"strings"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type InitCommand struct {
	Meta
}

func (c *InitCommand) Run(args []string) int {
	node, err := c.Meta.GetNode()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	session, err := mgo.Dial(node)
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
	return 0
}

func (c *InitCommand) Help() string {
	helpText := `
Usage: vault init [options]
  Initialize a new Vault server.
  This command connects to a Vault server and initializes it for the
  first time. This sets up the initial set of master keys and sets up the
  backend data store structure.
  This command can't be called on an already-initialized Vault.
General Options:
  -address=addr           The address of the Vault server.
  -ca-cert=path           Path to a PEM encoded CA cert file to use to
                          verify the Vault server SSL certificate.
  -ca-path=path           Path to a directory of PEM encoded CA cert files
                          to verify the Vault server SSL certificate. If both
                          -ca-cert and -ca-path are specified, -ca-path is used.
  -insecure               Do not verify TLS certificate. This is highly
                          not recommended. This is especially not recommended
                          for unsealing a vault.
Init Options:
  -key-shares=5           The number of key shares to split the master key
                          into.
  -key-threshold=3        The number of key shares required to reconstruct
                          the master key.
`
	return strings.TrimSpace(helpText)
}

func (c *InitCommand) Synopsis() string {
	return "Initilize a new replica set"
}
