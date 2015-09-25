package ovftool

import (
	"bytes"
	//gossh "code.google.com/p/go.crypto/ssh"
	gossh "golang.org/x/crypto/ssh"
	"fmt"
	"github.com/mitchellh/packer/communicator/ssh"
	"github.com/mitchellh/packer/packer"
	"io"
	"strings"
)

func (p *PostProcessor) Register() error {
	r, err := p.run(nil, "vim-cmd", "solo/registervm", p.vmxPath, p.config.VMName)
	if err != nil {
		return err
	}
	p.vmId = strings.TrimRight(r, "\n")
	return nil
}

func (p *PostProcessor) Unregister() error {
	return p.sh("vim-cmd", "vmsvc/unregister", p.vmId)
}

func (p *PostProcessor) connect() error {
	address := fmt.Sprintf("%s:%d", p.config.Host, p.config.SshPort)

	auth := []gossh.AuthMethod{
		gossh.Password(p.config.Password),
		gossh.KeyboardInteractive(
			ssh.PasswordKeyboardInteractive(p.config.Password)),
	}

	// TODO(dougm) KeyPath support
	sshConfig := &ssh.Config{
		Connection: ssh.ConnectFunc("tcp", address),
		SSHConfig: &gossh.ClientConfig{
			User: p.config.Username,
			Auth: auth,
		},
	}

	comm, err := ssh.New(address, sshConfig)
	if err != nil {
		return err
	}

	p.comm = comm
	return nil
}

func (p *PostProcessor) ssh(command string, stdin io.Reader) (*bytes.Buffer, error) {
	var stdout, stderr bytes.Buffer

	cmd := &packer.RemoteCmd{
		Command: command,
		Stdout:  &stdout,
		Stderr:  &stderr,
		Stdin:   stdin,
	}

	err := p.comm.Start(cmd)
	if err != nil {
		return nil, err
	}

	cmd.Wait()

	if cmd.ExitStatus != 0 {
		err = fmt.Errorf("'%s'\n\nStdout: %s\n\nStderr: %s",
			cmd.Command, stdout.String(), stderr.String())
		return nil, err
	}

	return &stdout, nil
}

func (p *PostProcessor) run(stdin io.Reader, args ...string) (string, error) {
	stdout, err := p.ssh(strings.Join(args, " "), stdin)
	if err != nil {
		return "", err
	}
	return stdout.String(), nil
}

func (p *PostProcessor) sh(args ...string) error {
	_, err := p.run(nil, args...)
	return err
}
