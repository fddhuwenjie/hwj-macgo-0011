package domain

import (
	"encoding/json"
)

// DeepCopyExperiment 深拷贝实验对象
func DeepCopyExperiment(src *Experiment) (*Experiment, error) {
	data, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}
	var dst Experiment
	if err := json.Unmarshal(data, &dst); err != nil {
		return nil, err
	}
	return &dst, nil
}

// DeepCopyRunPlan 深拷贝运行计划
func DeepCopyRunPlan(src *RunPlan) (*RunPlan, error) {
	data, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}
	var dst RunPlan
	if err := json.Unmarshal(data, &dst); err != nil {
		return nil, err
	}
	return &dst, nil
}

// 其他对象深拷贝可类似实现，为简洁略去，实际可复用通用方法
