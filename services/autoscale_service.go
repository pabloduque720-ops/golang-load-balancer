package services

import (
	"fmt"
	"time"

	"haproxy-manager/models"
	"haproxy-manager/storage"
)

func ValidateAutoScaleConfig(cfg models.AutoScaleConfig) error {
	if cfg.BalancerID <= 0 {
		return fmt.Errorf("balancer_id is required")
	}
	if cfg.HighThreshold <= 0 || cfg.HighThreshold > 100 {
		return fmt.Errorf("high_threshold must be between 0 and 100")
	}
	if cfg.LowThreshold < 0 || cfg.LowThreshold >= 100 {
		return fmt.Errorf("low_threshold must be between 0 and 100")
	}
	if cfg.LowThreshold >= cfg.HighThreshold {
		return fmt.Errorf("low_threshold must be lower than high_threshold")
	}
	if cfg.SampleSeconds <= 0 {
		return fmt.Errorf("sample_seconds must be greater than 0")
	}
	if cfg.SustainSeconds <= 0 {
		return fmt.Errorf("sustain_seconds must be greater than 0")
	}
	if cfg.MinInstances <= 0 {
		return fmt.Errorf("min_instances must be greater than 0")
	}
	if cfg.MaxInstances < cfg.MinInstances {
		return fmt.Errorf("max_instances must be greater than or equal to min_instances")
	}
	return nil
}

func SaveValidatedAutoScaleConfig(cfg models.AutoScaleConfig) error {
	if err := ValidateAutoScaleConfig(cfg); err != nil {
		return err
	}

	return storage.SaveAutoScaleConfig(cfg)
}

func GetAutoScaleConfig() (models.AutoScaleConfig, error) {
	return storage.LoadAutoScaleConfig()
}

func GetAutoScaleState() (models.AutoScaleState, error) {
	return storage.LoadAutoScaleState()
}

func CheckAutoScaleNow() (models.AutoScaleState, []map[string]interface{}, error) {
	cfg, err := storage.LoadAutoScaleConfig()
	if err != nil {
		return models.AutoScaleState{}, nil, err
	}

	if err := ValidateAutoScaleConfig(cfg); err != nil {
		return models.AutoScaleState{}, nil, err
	}

	avgCPU, details, err := GetConfiguredAverageCPU(cfg)
	if err != nil {
		return models.AutoScaleState{}, nil, err
	}

	servers, err := GetServersByBalancerID(cfg.BalancerID)
	if err != nil {
		return models.AutoScaleState{}, nil, err
	}

	state, err := storage.LoadAutoScaleState()
	if err != nil {
		return models.AutoScaleState{}, nil, err
	}

	state.LastCPUAverage = avgCPU
	state.ActiveInstances = len(servers)
	state.IsRunning = cfg.Enabled

	if avgCPU > cfg.HighThreshold {
		state.HighCounter += cfg.SampleSeconds
		state.LowCounter = 0

		if state.HighCounter >= cfg.SustainSeconds {
			state.LastAction = "scale_out_condition_detected"
			state.LastActionTime = time.Now().Format(time.RFC3339)
		} else {
			state.LastAction = "high_cpu_detected_waiting"
			state.LastActionTime = time.Now().Format(time.RFC3339)
		}
	} else if avgCPU < cfg.LowThreshold {
		state.LowCounter += cfg.SampleSeconds
		state.HighCounter = 0

		if state.LowCounter >= cfg.SustainSeconds {
			state.LastAction = "scale_in_condition_detected"
			state.LastActionTime = time.Now().Format(time.RFC3339)
		} else {
			state.LastAction = "low_cpu_detected_waiting"
			state.LastActionTime = time.Now().Format(time.RFC3339)
		}
	} else {
		state.HighCounter = 0
		state.LowCounter = 0
		state.LastAction = "within_thresholds"
		state.LastActionTime = time.Now().Format(time.RFC3339)
	}

	if err := storage.SaveAutoScaleState(state); err != nil {
		return models.AutoScaleState{}, nil, err
	}

	return state, details, nil
}