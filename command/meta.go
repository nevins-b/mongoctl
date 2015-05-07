package command

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/aocsolutions/mongoctl/builtin/consul"

	"github.com/mitchellh/cli"
)

// FlagSetFlags is an enum to define what flags are present in the
// default FlagSet returned by Meta.FlagSet.
type FlagSetFlags uint

const (
	FlagSetNone    FlagSetFlags = 0
	FlagSetServer  FlagSetFlags = 1 << iota
	FlagSetDefault              = FlagSetServer
	ec2MetadataURI              = "http://169.254.169.254/latest/meta-data/local-ipv4"
)

// Meta contains the meta-options and functionality that nearly every
// Vault command inherits.
type Meta struct {
	Ui cli.Ui

	// The things below can be set, but aren't common
	ForceAddress string  // Address to force for API clients
	ForceConfig  *Config // Force a config, don't load from disk

	// These are set by the command line flags.
	consulServer string
	consulKey    string
	consul       bool
	mongoServer  string
	consulAgent  *consul.Agent
	config       *Config
}

// Config loads the configuration and returns it. If the configuration
// is already loaded, it is returned.
func (m *Meta) Config() (*Config, error) {
	if m.config != nil {
		return m.config, nil
	}
	if m.ForceConfig != nil {
		return m.ForceConfig, nil
	}

	var err error
	m.config, err = LoadConfig("")
	if err != nil {
		return nil, err
	}

	return m.config, nil
}

// FlagSet returns a FlagSet with the common flags that every
// command implements. The exact behavior of FlagSet can be configured
// using the flags as the second parameter, for example to disable
// server settings on the commands that don't talk to a server.
func (m *Meta) FlagSet(n string, fs FlagSetFlags) *flag.FlagSet {
	f := flag.NewFlagSet(n, flag.ContinueOnError)

	// FlagSetServer tells us to enable the settings for selecting
	// the server information.
	if fs&FlagSetServer != 0 {
		f.StringVar(&m.consulKey, "consul-service", "mongodb", "")
		f.BoolVar(&m.consul, "consul", false, "")
		f.StringVar(&m.mongoServer, "mongo", "127.0.0.1:27017", "")
	}

	// Create an io.Writer that writes to our Ui properly for errors.
	// This is kind of a hack, but it does the job. Basically: create
	// a pipe, use a scanner to break it into lines, and output each line
	// to the UI. Do this forever.
	errR, errW := io.Pipe()
	errScanner := bufio.NewScanner(errR)
	go func() {
		for errScanner.Scan() {
			m.Ui.Error(errScanner.Text())
		}
	}()
	f.SetOutput(errW)

	return f
}

func (m *Meta) GetNode() (node string, err error) {
	if m.consul {
		nodes, err := m.consulAgent.GetService(
			m.consulKey,
			"",
		)
		if err != nil {
			return "", err
		}
		service := nodes[0]
		return fmt.Sprintf("%s:%d", service.ServiceAddress, service.ServicePort), nil
	}
	return m.mongoServer, nil
}

func (m *Meta) GetLocalIP() (ip string, err error) {
	resp, err := http.Get(ec2MetadataURI)
	if err != nil {
		return "", err
	}

	out, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", err
	}

	addr := string(out)
	_, err = net.ResolveIPAddr("ip", addr)
	if err != nil {
		return "", err
	}
	return addr, nil
}
