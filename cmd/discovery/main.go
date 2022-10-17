package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jaypipes/ghw/pkg/block"
	"github.com/kairos-io/go-tpm"
	"github.com/kairos-io/kairos/pkg/machine"
	"github.com/kairos-io/kcrypt/pkg/bus"
	"gopkg.in/yaml.v3"

	"github.com/mudler/go-pluggable"
)

func main() {
	if len(os.Args) >= 2 && bus.IsEventDefined(os.Args[1]) {
		checkErr(start())
	}

	pubhash, _ := tpm.GetPubHash()
	fmt.Print(pubhash)
}

func checkErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}

func getPass(server, label string) (string, error) {
	msg, err := tpm.Get(server, tpm.WithAdditionalHeader("label", label))
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
	return "", fmt.Errorf("pass for label not found")
}

// ❯ echo '{ "data": "{ \\"label\\": \\"LABEL\\" }"}' | sudo -E WSS_SERVER="http://localhost:8082/challenge" ./challenger "discovery.password"
func start() error {
	factory := pluggable.NewPluginFactory()

	connectionDetails := struct {
		Server string `yaml:"challenger_server"`
	}{}

	var server string
	d, err := machine.DotToYAML("/proc/cmdline")
	if err == nil { // best-effort
		yaml.Unmarshal(d, &connectionDetails) //nolint:errcheck
	}
	server = connectionDetails.Server
	if os.Getenv("WSS_SERVER") != "" {
		server = os.Getenv("WSS_SERVER")
	}

	// Input: bus.EventInstallPayload
	// Expected output: map[string]string{}
	factory.Add(bus.EventDiscoveryPassword, func(e *pluggable.Event) pluggable.EventResponse {
		if server == "" {
			return pluggable.EventResponse{
				Error: "no server configured",
			}
		}

		b := &block.Partition{}
		err := json.Unmarshal([]byte(e.Data), b)
		if err != nil {
			return pluggable.EventResponse{
				Error: fmt.Sprintf("failed reading partitions: %s", err.Error()),
			}
		}

		pass, err := getPass(server, b.Label)
		if err != nil {
			pass, err = getPass(server, b.Name)
			if err != nil {
				return pluggable.EventResponse{
					Error: fmt.Sprintf("failed getting pass: %s", err.Error()),
				}
			}
		}

		return pluggable.EventResponse{
			Data: pass,
		}
	})

	return factory.Run(pluggable.EventType(os.Args[1]), os.Stdin, os.Stdout)
}
