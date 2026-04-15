package services

import (
	"fmt"
	"strconv"
	"strings"

	"haproxy-manager/models"
)

func parseCPUStatLine(line string) (idle uint64, total uint64, err error) {
	fields := strings.Fields(line)
	if len(fields) < 8 || fields[0] != "cpu" {
		return 0, 0, fmt.Errorf("invalid /proc/stat cpu line")
	}

	values := make([]uint64, 0, len(fields)-1)
	for _, f := range fields[1:] {
		v, convErr := strconv.ParseUint(f, 10, 64)
		if convErr != nil {
			return 0, 0, convErr
		}
		values = append(values, v)
	}

	for _, v := range values {
		total += v
	}

	// idle + iowait
	idle = values[3]
	if len(values) > 4 {
		idle += values[4]
	}

	return idle, total, nil
}

func readRemoteCPUStat(cfg SSHConfig) (idle uint64, total uint64, err error) {
	cmd := "cat /proc/stat | head -n 1"
	output, err := RunSSHCommand(cfg, cmd)
	if err != nil {
		return 0, 0, err
	}

	line := strings.TrimSpace(output)
	return parseCPUStatLine(line)
}

func GetRemoteCPUUsage(cfg SSHConfig) (float64, error) {
	idle1, total1, err := readRemoteCPUStat(cfg)
	if err != nil {
		return 0, err
	}

	// pequeña espera remota no sirve; la espera la hacemos localmente con un segundo comando
	// para mantenerlo simple usamos sleep remoto entre lecturas
	output, err := RunSSHCommand(cfg, "sh -c 'cat /proc/stat | head -n 1; sleep 1; cat /proc/stat | head -n 1'")
	if err != nil {
		return 0, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return 0, fmt.Errorf("could not read two cpu samples")
	}

	idleA, totalA, err := parseCPUStatLine(strings.TrimSpace(lines[0]))
	if err != nil {
		return 0, err
	}

	idleB, totalB, err := parseCPUStatLine(strings.TrimSpace(lines[1]))
	if err != nil {
		return 0, err
	}

	// usamos la segunda medición doble si salió bien
	idle1 = idleA
	total1 = totalA

	deltaIdle := float64(idleB - idle1)
	deltaTotal := float64(totalB - total1)

	if deltaTotal <= 0 {
		return 0, fmt.Errorf("invalid cpu delta")
	}

	usage := 100.0 * (deltaTotal - deltaIdle) / deltaTotal
	if usage < 0 {
		usage = 0
	}
	if usage > 100 {
		usage = 100
	}

	return usage, nil
}

func GetBalancerAverageCPU(balancerID int, sshUser string) (float64, []map[string]interface{}, error) {
	servers, err := GetServersByBalancerID(balancerID)
	if err != nil {
		return 0, nil, err
	}

	if len(servers) == 0 {
		return 0, nil, fmt.Errorf("balancer %d has no servers", balancerID)
	}

	totalUsage := 0.0
	details := make([]map[string]interface{}, 0, len(servers))

	for _, server := range servers {
		cfg := SSHConfig{
			User: sshUser,
			Host: server.IPAddress,
		}

		usage, usageErr := GetRemoteCPUUsage(cfg)
		if usageErr != nil {
			return 0, nil, fmt.Errorf("error reading cpu from %s: %w", server.IPAddress, usageErr)
		}

		totalUsage += usage
		details = append(details, map[string]interface{}{
			"server_id":   server.ID,
			"name":        server.Name,
			"ip_address":  server.IPAddress,
			"port":        server.Port,
			"cpu_usage":   usage,
			"balancer_id": server.BalancerID,
		})
	}

	avg := totalUsage / float64(len(servers))
	return avg, details, nil
}

func GetConfiguredAverageCPU(cfg models.AutoScaleConfig) (float64, []map[string]interface{}, error) {
	if cfg.BalancerID <= 0 {
		return 0, nil, fmt.Errorf("autoscale balancer_id is required")
	}

	sshUser := "server1"
	if strings.TrimSpace(cfg.SSHUser) != "" {
		sshUser = cfg.SSHUser
	}

	return GetBalancerAverageCPU(cfg.BalancerID, sshUser)
}