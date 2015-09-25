package ovftool

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"github.com/mitchellh/packer/template/interpolate"
	"github.com/mitchellh/packer/common"
	"github.com/mitchellh/packer/helper/config"
	"github.com/mitchellh/packer/packer"
)


type Config struct {
	common.PackerConfig `mapstructure:",squash"`

	OVFtoolPath   string `mapstructure:"ovftool_path"`

	TargetType          string `mapstructure:"format"`
	TargetPath    string `mapstructure:"target"`
	Host          string `mapstructure:"remote_host"`
	ViPort        int    `mapstructure:"vi_port"`
	SshPort       int    `mapstructure:"ssh_port"`
	Username      string `mapstructure:"remote_username"`
	Password      string `mapstructure:"remote_password"`
	VMName        string `mapstructure:"vm_name"`

	CompressionLevel  	int    `mapstructure:"compression_level"`
        KeepInputArtifact bool   `mapstructure:"keep_input_artifact"`
	
	ctx interpolate.Context
}

type PostProcessor struct {
	config Config

	comm packer.Communicator
	vmId string
	vmxPath string
}

type OutputPathTemplate struct {
	ArtifactId string
	BuildName  string
	Provider   string
}

func (p *PostProcessor) Configure(raws ...interface{}) error {

  	err := config.Decode(&p.config, &config.DecodeOpts{
                Interpolate:        true,
                InterpolateContext: &p.config.ctx,
                InterpolateFilter: &interpolate.RenderFilter{
                        Exclude: []string{},
                },
     	}, raws...)

	errs := new(packer.MultiError)
        
	if err != nil {
		return err
	}
	//p.config.tpl.UserVars = p.config.ctx.UserVariables
	
	if err = interpolate.Validate(p.config.TargetPath, &p.config.ctx); err != nil {
                errs = packer.MultiErrorAppend(
                        errs, fmt.Errorf("Error parsing target template: %s", err))
                }

	if p.config.OVFtoolPath == "" {
		p.config.OVFtoolPath = "ovftool"
	}
	
	if p.config.TargetType == "" {
		p.config.TargetType = "ova"
	}
	if p.config.TargetPath == "" {
			p.config.TargetPath = "packer_build_provider"
			if p.config.TargetType == "ova" {
				p.config.TargetPath += ".ova"
			}
	}


	if p.config.ViPort == 0 {
		p.config.ViPort = 443
	}

	if p.config.SshPort == 0 {
                p.config.SshPort = 22
        }

	if p.config.Username == "" {
		p.config.Username = "root"
	}

	if !(p.config.TargetType == "ovf" || p.config.TargetType == "ova") {
		errs = packer.MultiErrorAppend(
			errs, fmt.Errorf("Invalid target type. Only 'ovf' or 'ova' are allowed."))
	}

	if !(p.config.CompressionLevel >= 0 && p.config.CompressionLevel <= 9) {
		errs = packer.MultiErrorAppend(
			errs, fmt.Errorf("Invalid compression level. Must be between 1 and 9, or 0 for no compression."))
	}

	if len(errs.Errors) > 0 {
		return errs
	}

	if p.config.Host == "" {
		return fmt.Errorf("ovftool post processor: host parameter is required")
	}

	if p.config.Password == "" {
		return fmt.Errorf("ovftool post processor: password parameter is required")
	}

	if p.config.VMName == "" {
		return fmt.Errorf("ovftool post processor: vm_name parameter is required")
	}


	return nil
}

func (p *PostProcessor) PostProcess(ui packer.Ui, artifact packer.Artifact) (packer.Artifact, bool, error) {
        keep := p.config.KeepInputArtifact

	if artifact.BuilderId() != "mitchellh.vmware-esx" {
		return nil, keep, fmt.Errorf("ovftool post-processor can only be used on VMware ESX builds: %s", artifact.BuilderId())
	}

	// Debug information
       //ui.Message(fmt.Sprintf("[DEBUG] BuildName=%s,Provider=%s",buildname,provider))
       ui.Message(fmt.Sprintf("[DEBUG] BuildName=%s,BuildType=%s,TemplatePath=%s",p.config.ctx.BuildName,p.config.ctx.BuildType,p.config.ctx.TemplatePath))


	for _, f := range artifact.Files() {
		if strings.HasSuffix(f, ".vmx") {
			p.vmxPath = f
		}
	}

	if p.vmxPath == "" {
		return nil, keep, fmt.Errorf("No .vmx file in artifact")
	}

	ui.Say( "Registering VM...")

	ui.Message(fmt.Sprintf("ESXi host address %s:%d", p.config.Host, p.config.SshPort))

	err := p.connect()
	if err != nil {
		return nil, keep, err
	}

	err = p.Register()
	if err != nil {
		return nil, keep, err
	}
	defer p.Unregister()
	
	ui.Say( "Exporting VM...")

	// build the arguments
	args := []string{
		"--targetType=" + p.config.TargetType,
		"--acceptAllEulas",
		"--noSSLVerify",
	}

	// append --compression, if it is set
	if p.config.CompressionLevel > 0 {
		args = append(args, fmt.Sprintf("--compress=%d", p.config.CompressionLevel))
	}

	var stdout, stderr bytes.Buffer
	
	source := fmt.Sprintf( "vi://%s:%s@%s/%s", p.config.Username, p.config.Password, p.config.Host, p.config.VMName)


	args = append(args, source, p.config.TargetPath)

	cmd := exec.Command( p.config.OVFtoolPath,args...)

	//ui.Message(fmt.Sprintf( "source = %s, cmd = %v",source,cmd.Args))
	
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
	  p.Unregister()
		return nil, keep, fmt.Errorf("Unable to execute ovftool:\n== STDOUT ==\n%s== STDERR ==\n%s", stdout.String(), stderr.String())
	}

	return &Artifact{ ova_file: p.config.TargetPath , id: p.config.TargetType }, keep, nil
}
