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
	out := fmt.Sprintf("%v", result)
	c.Ui.Output(out)
	return 0
}

func (c *StatusCommand) Help() string {
	helpText := `
Usage: mongoctl init [options]
  Initialize a new MongoDB Cluster
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
`
	return strings.TrimSpace(helpText)
}

func (c *StatusCommand) Synopsis() string {
	return "Initilize a new replica set"
}
