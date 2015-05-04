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
	var priority int
	var hidden, arbitrator bool
	var addr, username string
	flags := c.Meta.FlagSet("add", FlagSetDefault)
	flags.Usage = func() { c.Ui.Error(c.Help()) }
	flags.IntVar(&priority, "priority", 1, "")
	flags.StringVar(&addr, "addr", "127.0.0.1:27018", "")
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
		Host:        addr,
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
	out := fmt.Sprintf("%v", result)
	c.Ui.Output(out)
	return 0
}

func (c *AddCommand) Help() string {
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

func (c *AddCommand) Synopsis() string {
	return "Add a node to an existing mongo cluster"
}
