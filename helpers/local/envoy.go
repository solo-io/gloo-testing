package localhelpers

import (
	"os"
	"os/exec"
	"path/filepath"

	"io/ioutil"
)

const defualtEnvoyDockerImage = "soloio/envoy:v0.1.2"

const envoyfconfig = `
node:
 cluster: ingress
 id: testnode

static_resources:
  clusters:
  - name: xds_cluster
    connect_timeout: 5.000s
    hosts:
    - socket_address:
        address: localhost
        port_value: 8081
    http2_protocol_options: {}
    type: STRICT_DNS

dynamic_resources:
  ads_config:
    api_type: GRPC
    cluster_names:
    - xds_cluster
  cds_config:
    ads: {}
  lds_config:
    ads: {}
  
admin:
  access_log_path: /dev/null
  address:
    socket_address:
      address: 0.0.0.0
      port_value: 19000

`

type EnvoyInstance struct {
	envoypath    string
	envoycfgpath string
	tmpdir       string
	cmd          *exec.Cmd
}

func NewEnvoyInstance() (*EnvoyInstance, error) {
	envoypath := os.Getenv("ENVOY_BINARY")

	// try to grab one form docker...
	tmpdir, err := ioutil.TempDir(os.Getenv("HELPER_TMP"), "envoy")
	if err != nil {
		return nil, err
	}

	envoyconfigyaml := filepath.Join(tmpdir, "envoyconfig.yaml")

	ioutil.WriteFile(envoyconfigyaml, []byte(envoyfconfig), 0644)

	if envoypath != "" {
		return &EnvoyInstance{
			envoypath:    envoypath,
			envoycfgpath: envoyconfigyaml,
			tmpdir:       tmpdir,
		}, nil
	}

	bash := `
set -ex
CID=$(docker run -d  soloio/envoy:v0.1.2 /bin/bash -c exit)
docker cp $CID:/usr/local/bin/envoy .
docker rm $CID
    `
	scriptfile := filepath.Join(tmpdir, "getenvoy.sh")

	ioutil.WriteFile(scriptfile, []byte(bash), 0755)

	cmd := exec.Command("bash", scriptfile)
	cmd.Dir = tmpdir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return &EnvoyInstance{
		envoypath:    filepath.Join(tmpdir, "envoy"),
		envoycfgpath: envoyconfigyaml,
		tmpdir:       tmpdir,
	}, nil
}

func (ei *EnvoyInstance) Run() error {
	cmd := exec.Command(ei.envoypath, "-c", ei.envoycfgpath, "--v2-config-only")
	cmd.Dir = ei.tmpdir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		return err
	}
	ei.cmd = cmd
	return nil
}

func (ei *EnvoyInstance) Binary() string {
	return ei.envoypath
}

func (ei *EnvoyInstance) Clean() error {
	if ei.cmd != nil {
		ei.cmd.Process.Kill()
		ei.cmd.Wait()
	}
	if ei.tmpdir != "" {
		defer os.RemoveAll(ei.tmpdir)

	}
	return nil
}