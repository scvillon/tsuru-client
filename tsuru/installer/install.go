// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tsuru/config"
	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru-client/tsuru/admin"
	"github.com/tsuru/tsuru-client/tsuru/client"
	"github.com/tsuru/tsuru-client/tsuru/installer/dm"
	"github.com/tsuru/tsuru/cmd"
)

var (
	defaultTsuruInstallConfig = &TsuruInstallConfig{
		DockerMachineConfig: dm.DefaultDockerMachineConfig,
		ComponentsConfig:    NewInstallConfig(dm.DefaultDockerMachineConfig.Name),
		CoreHosts:           1,
		AppsHosts:           1,
		DedicatedAppsHosts:  false,
		CoreDriversOpts:     make(map[string][]interface{}),
	}
)

type TsuruInstallConfig struct {
	*dm.DockerMachineConfig
	*ComponentsConfig
	CoreHosts          int
	CoreDriversOpts    map[string][]interface{}
	AppsHosts          int
	DedicatedAppsHosts bool
	AppsDriversOpts    map[string][]interface{}
}

type Install struct {
	fs     *gnuflag.FlagSet
	config string
}

func (c *Install) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "install",
		Usage: "install [--config/-c config_file]",
		Desc: `Installs Tsuru and It's components as containers on hosts provisioned
with docker machine drivers.

The [[--config]] parameter is the path to a .yml file containing the installation
configuration. If not provided, Tsuru will be installed into a VirtualBox VM for
experimentation.

The following is an example of installation configuration to install Tsuru on
Amazon EC2:

==========
name: tsuru-ec2
driver:
    name: amazonec2
    options:
        amazonec2-access-key: myAmazonAccessKey
        amazonec2-secret-key: myAmazonSecretKey
        amazonec2-vpc-id: vpc-abc1234
        amazonec2-subnet-id: subnet-abc1234
==========

Available configuration parameters:

- name
Name of the installation.

- docker-hub-mirror
Url of a docker hub mirror used to fetch the components docker images.

- ca-path
A path to a directory containing a ca.pem and ca-key.pem files that are going to be used to sign certificates used by docker and docker registry.
If not set, a CA will be created, copied to every host provisioned and used to sign the certificates.

- hosts:core:size
Number of machines to be provisioned and used to host tsuru core components.

- hosts:core:driver:options
Driver parameters specific to the core hosts can be set on this namespace. The format is: <driver-param>>: ["value1", "value2"]. Each
host will use one value from the list. Refer to the driver configuration for more information on what parameter are available.

- hosts:apps:size
Number of machines to be provisioned and used to host tsuru applications.

- hosts:apps:dedicated
Boolean to indicated if the installer should not reuse the machines created for
the core components.

- hosts:apps:driver:options
Driver parameters specific to the applications hosts can be set on this namespace. The format is: <driver-param>>: ["value1", "value2"]. Each
host will use one value from the list. Refer to the driver configuration for more information on what parameter are available.

- driver
Under this namespace lies all the docker machine driver configuration.

- driver:name
Name of the driver to be used by the installer. This can be any core or 3rd party driver supported by docker machine. If a 3rd party driver name is used, it's binary must be available on the user path.

- driver:options
Under this namespace every driver parameters can be set. Refer to the driver configuration for more information on what parameter are available.
`,
		MinArgs: 0,
	}
}

func (c *Install) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("install", gnuflag.ExitOnError)
		c.fs.StringVar(&c.config, "c", "", "Configuration file")
		c.fs.StringVar(&c.config, "config", "", "Configuration file")
	}
	return c.fs
}

func (c *Install) Run(context *cmd.Context, cli *cmd.Client) error {
	context.RawOutput()
	config, err := parseConfigFile(c.config)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, "Running pre-install checks...\n")
	err = c.PreInstallChecks(config)
	if err != nil {
		return fmt.Errorf("pre-install checks failed: %s", err)
	}
	dockerMachine, err := dm.NewDockerMachine(config.DockerMachineConfig)
	if err != nil {
		return fmt.Errorf("failed to create docker machine: %s", err)
	}
	defer dockerMachine.Close()
	config.CoreDriversOpts[config.DriverName+"-open-port"] = []interface{}{strconv.Itoa(defaultTsuruAPIPort)}
	coreMachines, err := ProvisionMachines(dockerMachine, config.CoreHosts, config.CoreDriversOpts)
	if err != nil {
		return fmt.Errorf("failed to provision components machines: %s", err)
	}
	cluster, err := NewSwarmCluster(coreMachines, len(coreMachines))
	if err != nil {
		return fmt.Errorf("failed to setup swarm cluster: %s", err)
	}
	for _, component := range TsuruComponents {
		fmt.Fprintf(context.Stdout, "Installing %s\n", component.Name())
		errInstall := component.Install(cluster, config.ComponentsConfig)
		if errInstall != nil {
			return fmt.Errorf("error installing %s: %s", component.Name(), errInstall)
		}
		fmt.Fprintf(context.Stdout, "%s successfully installed!\n", component.Name())
	}
	appsMachines, err := ProvisionPool(dockerMachine, config, coreMachines)
	if err != nil {
		return err
	}
	var nodesAddr []string
	for _, m := range appsMachines {
		nodesAddr = append(nodesAddr, m.GetPrivateAddress())
	}
	fmt.Fprintf(context.Stdout, "Bootstrapping Tsuru API...")
	opts := TsuruSetupOptions{
		Login:      config.ComponentsConfig.RootUserEmail,
		Password:   config.ComponentsConfig.RootUserPassword,
		Target:     fmt.Sprintf("http://%s:%d", cluster.GetManager().IP, defaultTsuruAPIPort),
		TargetName: config.ComponentsConfig.TargetName,
		NodesAddr:  nodesAddr,
	}
	err = SetupTsuru(opts)
	if err != nil {
		return fmt.Errorf("Error bootstrapping tsuru: %s", err)
	}
	fmt.Fprintf(context.Stdout, "Applying iptables workaround for docker 1.12...\n")
	for _, m := range coreMachines {
		_, err = m.RunSSHCommand("PATH=$PATH:/usr/sbin/:/usr/local/sbin; sudo iptables -D DOCKER-ISOLATION -i docker_gwbridge -o docker0 -j DROP")
		if err != nil {
			fmt.Fprintf(context.Stderr, "Failed to apply iptables rule: %s. Maybe it is not needed anymore?\n", err)
		}
		_, err = m.RunSSHCommand("PATH=$PATH:/usr/sbin/:/usr/local/sbin; sudo iptables -D DOCKER-ISOLATION -i docker0 -o docker_gwbridge -j DROP")
		if err != nil {
			fmt.Fprintf(context.Stderr, "Failed to apply iptables rule: %s. Maybe it is not needed anymore?\n", err)
		}
	}
	fmt.Fprint(context.Stdout, "--- Installation Overview ---\n")
	fmt.Fprint(context.Stdout, "Core Hosts: \n"+buildClusterTable(cluster).String())
	fmt.Fprint(context.Stdout, "Core Components: \n"+buildComponentsTable(TsuruComponents, cluster).String())
	fmt.Fprintln(context.Stdout, "Apps Hosts:")
	nodeList := &admin.ListNodesCmd{}
	nodeList.Run(context, cli)
	fmt.Fprintln(context.Stdout, "Apps:")
	appList := &client.AppList{}
	appList.Run(context, cli)
	machineIndex := make(map[string]*dm.Machine)
	allMachines := append(coreMachines, appsMachines...)
	for _, m := range allMachines {
		machineIndex[m.Name] = m
	}
	var uniqueMachines []*dm.Machine
	for _, v := range machineIndex {
		uniqueMachines = append(uniqueMachines, v)
	}
	err = addInstallHosts(uniqueMachines, cli)
	if err != nil {
		return fmt.Errorf("failed to register hosts: %s", err)
	}
	return nil
}

func addInstallHosts(machines []*dm.Machine, client *cmd.Client) error {
	path, err := cmd.GetURLVersion("1.3", "/install/hosts")
	if err != nil {
		return err
	}
	for _, m := range machines {
		rawDriver, err := json.Marshal(m.Driver)
		if err != nil {
			return err
		}
		privateKey, err := ioutil.ReadFile(m.GetSSHKeyPath())
		if err != nil {
			fmt.Printf("failed to read private ssh key file: %s", err)
		}
		caCert, err := ioutil.ReadFile(filepath.Join(m.CAPath, "ca.pem"))
		if err != nil {
			fmt.Printf("failed to read ca file: %s", err)
		}
		caPrivateKey, err := ioutil.ReadFile(filepath.Join(m.CAPath, "ca-key.pem"))
		if err != nil {
			fmt.Printf("failed to read ca private key file: %s", err)
		}
		v := url.Values{}
		v.Set("driver", string(rawDriver))
		v.Set("name", m.Name)
		v.Set("driverName", m.DriverName)
		v.Set("sshPrivateKey", string(privateKey))
		v.Set("caCert", string(caCert))
		v.Set("caPrivateKey", string(caPrivateKey))
		body := strings.NewReader(v.Encode())
		request, err := http.NewRequest("POST", path, body)
		if err != nil {
			return err
		}
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		_, err = client.Do(request)
		if err != nil {
			return err
		}
	}
	return nil
}

func ProvisionPool(p dm.MachineProvisioner, config *TsuruInstallConfig, hosts []*dm.Machine) ([]*dm.Machine, error) {
	if config.DedicatedAppsHosts {
		return ProvisionMachines(p, config.AppsHosts, config.AppsDriversOpts)
	}
	if config.AppsHosts > len(hosts) {
		poolMachines, err := ProvisionMachines(p, config.AppsHosts-len(hosts), config.AppsDriversOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to provision pool hosts: %s", err)
		}
		return append(poolMachines, hosts...), nil
	}
	return hosts[:config.AppsHosts], nil
}

func ProvisionMachines(p dm.MachineProvisioner, numMachines int, configs map[string][]interface{}) ([]*dm.Machine, error) {
	var machines []*dm.Machine
	for i := 0; i < numMachines; i++ {
		opts := make(dm.DriverOpts)
		for k, v := range configs {
			idx := i % len(v)
			opts[k] = v[idx]
		}
		m, err := p.ProvisionMachine(opts)
		if err != nil {
			return nil, fmt.Errorf("failed to provision machines: %s", err)
		}
		machines = append(machines, m)
	}
	return machines, nil
}

func (c *Install) PreInstallChecks(config *TsuruInstallConfig) error {
	exists, err := cmd.CheckIfTargetLabelExists(config.Name)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("tsuru target \"%s\" already exists", config.Name)
	}
	return nil
}

func buildClusterTable(cluster ServiceCluster) *cmd.Table {
	t := cmd.NewTable()
	t.Headers = cmd.Row{"IP", "State", "Manager"}
	t.LineSeparator = true
	nodes, err := cluster.ClusterInfo()
	if err != nil {
		t.AddRow(cmd.Row{fmt.Sprintf("failed to retrieve cluster info: %s", err)})
	}
	for _, n := range nodes {
		t.AddRow(cmd.Row{n.IP, n.State, strconv.FormatBool(n.Manager)})
	}
	return t
}

func buildComponentsTable(components []TsuruComponent, cluster ServiceCluster) *cmd.Table {
	t := cmd.NewTable()
	t.Headers = cmd.Row{"Component", "Ports", "Replicas"}
	t.LineSeparator = true
	for _, component := range components {
		info, err := component.Status(cluster)
		if err != nil {
			t.AddRow(cmd.Row{component.Name(), "?", fmt.Sprintf("%s", err)})
			continue
		}
		row := cmd.Row{component.Name(),
			strings.Join(info.Ports, ","),
			strconv.Itoa(info.Replicas),
		}
		t.AddRow(row)
	}
	return t
}

type Uninstall struct {
	fs     *gnuflag.FlagSet
	config string
}

func (c *Uninstall) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "uninstall",
		Usage:   "uninstall [--config/-c config_file]",
		Desc:    "Uninstalls Tsuru and It's components.",
		MinArgs: 0,
	}
}

func (c *Uninstall) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("uninstall", gnuflag.ExitOnError)
		c.fs.StringVar(&c.config, "c", "", "Configuration file")
		c.fs.StringVar(&c.config, "config", "", "Configuration file")
	}
	return c.fs
}

func (c *Uninstall) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	config, err := parseConfigFile(c.config)
	if err != nil {
		fmt.Fprintf(context.Stderr, "Failed to read configuration file: %s\n", err)
		return err
	}
	d, err := dm.NewDockerMachine(config.DockerMachineConfig)
	if err != nil {
		fmt.Fprintf(context.Stderr, "Failed to delete machine: %s\n", err)
		return err
	}
	defer d.Close()
	err = d.DeleteAll()
	if err != nil {
		fmt.Fprintf(context.Stderr, "Failed to delete machines: %s\n", err)
		return err
	}
	fmt.Fprintln(context.Stdout, "Machines successfully removed!")
	api := TsuruAPI{}
	err = api.Uninstall(config.Name)
	if err != nil {
		fmt.Fprintf(context.Stderr, "Failed to uninstall tsuru API: %s\n", err)
		return err
	}
	fmt.Fprintf(context.Stdout, "Uninstall finished successfully!\n")
	return nil
}

func parseConfigFile(file string) (*TsuruInstallConfig, error) {
	installConfig := defaultTsuruInstallConfig
	if file == "" {
		return installConfig, nil
	}
	err := config.ReadConfigFile(file)
	if err != nil {
		return nil, err
	}
	driverName, err := config.GetString("driver:name")
	if err == nil {
		installConfig.DriverName = driverName
	}
	name, err := config.GetString("name")
	if err == nil {
		installConfig.Name = name
	}
	hub, err := config.GetString("docker-hub-mirror")
	if err == nil {
		installConfig.DockerHubMirror = hub
	}
	driverOpts := make(dm.DriverOpts)
	opts, _ := config.Get("driver:options")
	if opts != nil {
		for k, v := range opts.(map[interface{}]interface{}) {
			switch k := k.(type) {
			case string:
				driverOpts[k] = v
			}
		}
		installConfig.DriverOpts = driverOpts
	}
	caPath, err := config.GetString("ca-path")
	if err == nil {
		installConfig.CAPath = caPath
	}
	cHosts, err := config.GetInt("hosts:core:size")
	if err == nil {
		installConfig.CoreHosts = cHosts
	}
	pHosts, err := config.GetInt("hosts:apps:size")
	if err == nil {
		installConfig.AppsHosts = pHosts
	}
	dedicated, err := config.GetBool("hosts:apps:dedicated")
	if err == nil {
		installConfig.DedicatedAppsHosts = dedicated
	}
	opts, _ = config.Get("hosts:core:driver:options")
	if opts != nil {
		installConfig.CoreDriversOpts, err = parseDriverOptsSlice(opts)
		if err != nil {
			return nil, err
		}
	}
	opts, _ = config.Get("hosts:apps:driver:options")
	if opts != nil {
		installConfig.AppsDriversOpts, err = parseDriverOptsSlice(opts)
		if err != nil {
			return nil, err
		}
	}
	installConfig.ComponentsConfig = NewInstallConfig(installConfig.Name)
	return installConfig, nil
}

func parseDriverOptsSlice(opts interface{}) (map[string][]interface{}, error) {
	unparsed, ok := opts.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to parse opts: %+v", opts)
	}
	parsedOpts := make(map[string][]interface{})
	if opts != nil {
		for k, v := range unparsed {
			switch k := k.(type) {
			case string:
				l, ok := v.([]interface{})
				if ok {
					parsedOpts[k] = l
				} else {
					parsedOpts[k] = []interface{}{v}
				}
			}
		}
	}
	return parsedOpts, nil
}

type InstallHostList struct{}

type installHost struct {
	Name          string
	DriverName    string
	Driver        map[string]interface{}
	SSHPrivateKey string
}

func (c *InstallHostList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "install-host-list",
		Usage:   "install-host-list",
		Desc:    "List hosts created and registered by the installer.",
		MinArgs: 0,
		MaxArgs: 0,
	}
}

func (c *InstallHostList) Flags() *gnuflag.FlagSet {
	return gnuflag.NewFlagSet("install-host-list", gnuflag.ExitOnError)
}

func (c *InstallHostList) Run(context *cmd.Context, cli *cmd.Client) error {
	url, err := cmd.GetURLVersion("1.3", "/install/hosts")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	response, err := cli.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	return c.Show(body, context)
}

func (c *InstallHostList) Show(result []byte, context *cmd.Context) error {
	var hosts []installHost
	err := json.Unmarshal(result, &hosts)
	if err != nil {
		return err
	}
	dockerMachine, err := dm.NewTempDockerMachine()
	if err != nil {
		return err
	}
	defer dockerMachine.Close()
	table := cmd.NewTable()
	table.LineSeparator = true
	table.Headers = cmd.Row([]string{"Name", "Driver Name", "State", "Driver"})
	for _, h := range hosts {
		driver, err := json.MarshalIndent(h.Driver, "", " ")
		if err != nil {
			return err
		}
		host, err := dockerMachine.NewHost(h.DriverName, h.SSHPrivateKey, h.Driver)
		if err != nil {
			return err
		}
		state, err := host.Driver.GetState()
		var stateStr string
		if err != nil {
			stateStr = err.Error()
		} else {
			stateStr = state.String()
		}
		table.AddRow(cmd.Row([]string{h.Name, h.DriverName, stateStr, string(driver)}))
	}
	context.Stdout.Write(table.Bytes())
	return nil
}

type InstallSSH struct{}

func (c *InstallSSH) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "install-ssh",
		Usage:   "install-ssh <hostname> [arg...]",
		Desc:    "Log into or run a command on a host with SSH.",
		MinArgs: 1,
	}
}

func (c *InstallSSH) Flags() *gnuflag.FlagSet {
	return gnuflag.NewFlagSet("install-ssh", gnuflag.ExitOnError)
}

func (c *InstallSSH) Run(context *cmd.Context, cli *cmd.Client) error {
	hostName := context.Args[0]
	url, err := cmd.GetURLVersion("1.3", "/install/hosts/"+hostName)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	response, err := cli.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	var ih *installHost
	err = json.NewDecoder(response.Body).Decode(&ih)
	if err != nil {
		return err
	}
	dockerMachine, err := dm.NewTempDockerMachine()
	if err != nil {
		return err
	}
	defer dockerMachine.Close()
	h, err := dockerMachine.NewHost(ih.DriverName, ih.SSHPrivateKey, ih.Driver)
	if err != nil {
		return err
	}
	sshClient, err := h.CreateSSHClient()
	if err != nil {
		return fmt.Errorf("failed to create ssh client: %s", err)
	}
	sshArgs := []string{}
	if len(context.Args) > 1 {
		sshArgs = context.Args[1:]
	}
	return sshClient.Shell(sshArgs...)
}
