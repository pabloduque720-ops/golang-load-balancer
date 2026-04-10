package services

import (
	"fmt"
	"os/exec"
)

type SSHConfig struct {
	User       string
	Host       string
	RemotePath string
}

func RunCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func RunSSHCommand(cfg SSHConfig, command string) (string, error) {
	target := fmt.Sprintf("%s@%s", cfg.User, cfg.Host)
	return RunCommand("ssh", target, command)
}

func CopyFileBySCP(cfg SSHConfig, localPath string) (string, error) {
	target := fmt.Sprintf("%s@%s:%s", cfg.User, cfg.Host, cfg.RemotePath)
	return RunCommand("scp", localPath, target)
}

func ApplyRemoteHAProxyConfig(cfg SSHConfig, localConfigPath string) error {
	_, err := CopyFileBySCP(cfg, localConfigPath)
	if err != nil {
		return fmt.Errorf("error copying file by scp: %w", err)
	}

	moveCmd := fmt.Sprintf("sudo mv %s/haproxy.cfg /etc/haproxy/haproxy.cfg", cfg.RemotePath)
	if _, err := RunSSHCommand(cfg, moveCmd); err != nil {
		return fmt.Errorf("error moving haproxy.cfg on remote server: %w", err)
	}

	validateCmd := "sudo haproxy -c -f /etc/haproxy/haproxy.cfg"
	if output, err := RunSSHCommand(cfg, validateCmd); err != nil {
		return fmt.Errorf("haproxy config validation failed: %v - output: %s", err, output)
	}

	reloadCmd := "sudo systemctl reload haproxy"
	if output, err := RunSSHCommand(cfg, reloadCmd); err != nil {
		return fmt.Errorf("error reloading haproxy: %v - output: %s", err, output)
	}

	return nil
}