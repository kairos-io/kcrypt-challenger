package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jaypipes/ghw/pkg/block"
	"github.com/kairos-io/go-tpm"
	"github.com/kairos-io/kairos/pkg/machine"
	"github.com/kairos-io/kcrypt/pkg/bus"
	"github.com/mudler/go-pluggable"
	"gopkg.in/yaml.v1"
)

// If this file exists, it will be used as configuration
const DefaultConfigLocation = "/oem/kcrypt-challenger.conf"

type Config struct {
	Server string `yaml:"challenger_server"`
}

type Client struct {
	Config Config
}

func NewClient() (*Client, error) {
	conf, err := GetConfiguration(DefaultConfigLocation)
	if err != nil {
		return nil, err
	}

	return &Client{Config: conf}, nil
}

// ❯ echo '{ "data": "{ \\"label\\": \\"LABEL\\" }"}' | sudo -E WSS_SERVER="http://localhost:8082/challenge" ./challenger "discovery.password"
func (c *Client) Start() error {
	factory := pluggable.NewPluginFactory()

	// Input: bus.EventInstallPayload
	// Expected output: map[string]string{}
	factory.Add(bus.EventDiscoveryPassword, func(e *pluggable.Event) pluggable.EventResponse {

		b := &block.Partition{}
		err := json.Unmarshal([]byte(e.Data), b)
		if err != nil {
			return pluggable.EventResponse{
				Error: fmt.Sprintf("failed reading partitions: %s", err.Error()),
			}
		}

		pass, err := c.waitPass(b, 30)
		if err != nil {
			return pluggable.EventResponse{
				Error: fmt.Sprintf("failed getting pass: %s", err.Error()),
			}
		}

		return pluggable.EventResponse{
			Data: pass,
		}
	})

	return factory.Run(pluggable.EventType(os.Args[1]), os.Stdin, os.Stdout)
}

func (c *Client) waitPass(p *block.Partition, attempts int) (pass string, err error) {
	for tries := 0; tries < attempts; tries++ {
		if c.Config.Server == "" {
			err = fmt.Errorf("no server configured")
			continue
		}

		pass, err = c.getPass(c.Config.Server, p)
		if pass != "" || err == nil {
			return pass, err
		}
		time.Sleep(1 * time.Second)
	}
	return
}

func (c *Client) getPass(server string, partition *block.Partition) (string, error) {
	msg, err := tpm.Get(server,
		tpm.WithAdditionalHeader("label", partition.Label),
		tpm.WithAdditionalHeader("name", partition.Name),
		tpm.WithAdditionalHeader("uuid", partition.UUID))
	if err != nil {
		return "", err
	}
	result := map[string]interface{}{}
	err = json.Unmarshal(msg, &result)
	if err != nil {
		return "", err
	}
	p, ok := result["passphrase"]
	if ok {
		return fmt.Sprint(p), nil
	}
	return "", fmt.Errorf("pass for partition not found")
}

// Combines configuration from cmdline, environment variables and the
// DefaultConfigLocation file into one struct.
func GetConfiguration(configFile string) (Config, error) {
	result := Config{}

	if err := getConfigurationFromCmdLine(&result); err != nil {
		return result, err
	}
	getConfigurationFromEnv(&result)
	if err := getConfigurationFromFile(&result, configFile); err != nil {
		return result, err
	}

	return result, nil
}

func getConfigurationFromCmdLine(c *Config) error {
	// best-effort
	d, err := machine.DotToYAML("/proc/cmdline")
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(d, c)
	if err != nil {
		return err
	}
	return nil
}

func getConfigurationFromEnv(c *Config) {
	if os.Getenv("WSS_SERVER") != "" {
		c.Server = os.Getenv("WSS_SERVER")
	}
}

func getConfigurationFromFile(c *Config, configFile string) error {
	confData, err := os.ReadFile(configFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	return yaml.Unmarshal(confData, &c)
}
