package domain

import (
	"encoding/json"
	"time"
)

// ExperimentStatus 定义实验状态
type ExperimentStatus string

const (
	ExperimentDraft     ExperimentStatus = "draft"
	ExperimentFrozen    ExperimentStatus = "frozen"
	ExperimentQueued    ExperimentStatus = "queued"
	ExperimentRunning   ExperimentStatus = "running"
	ExperimentSealed    ExperimentStatus = "sealed"
	ExperimentPublished ExperimentStatus = "published"
	ExperimentFailed    ExperimentStatus = "failed"
)

// Experiment 实验聚合根
type Experiment struct {
	ID                     string            `json:"id"`
	Name                   string            `json:"name"`
	Description            string            `json:"description"`
	CurrentParamVersionID  string            `json:"current_param_version_id"`
	CurrentInputSnapshotID string            `json:"current_input_snapshot_id"`
	Status                 ExperimentStatus  `json:"status"`
	Version                int               `json:"version"`
	CreatedAt              time.Time         `json:"created_at"`
	UpdatedAt              time.Time         `json:"updated_at"`
	Metadata               map[string]string `json:"metadata,omitempty"`
}

// NewExperiment 创建新实验，初始状态为草稿
func NewExperiment(id, name, description string) *Experiment {
	now := time.Now().UTC()
	return &Experiment{
		ID:          id,
		Name:        name,
		Description: description,
		Status:      ExperimentDraft,
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
		Metadata:    make(map[string]string),
	}
}

// CanModify 判断实验是否可修改（仅草稿或冻结前可修改）
func (e *Experiment) CanModify() bool {
	return e.Status == ExperimentDraft || e.Status == ExperimentFrozen
}

// Freeze 冻结实验，锁定参数和输入快照
func (e *Experiment) Freeze(paramVersionID, inputSnapshotID string) error {
	if e.Status != ExperimentDraft {
		return ErrInvalidStatus
	}
	if paramVersionID == "" || inputSnapshotID == "" {
		return ErrInvalidParameter
	}
	e.CurrentParamVersionID = paramVersionID
	e.CurrentInputSnapshotID = inputSnapshotID
	e.Status = ExperimentFrozen
	e.UpdatedAt = time.Now().UTC()
	e.Version++
	return nil
}

// Queue 将实验排队，表示计划已就绪
func (e *Experiment) Queue() error {
	if e.Status != ExperimentFrozen {
		return ErrInvalidStatus
	}
	e.Status = ExperimentQueued
	e.UpdatedAt = time.Now().UTC()
	e.Version++
	return nil
}

// MarkRunning 标记实验正在执行
func (e *Experiment) MarkRunning() error {
	if e.Status != ExperimentQueued {
		return ErrInvalidStatus
	}
	e.Status = ExperimentRunning
	e.UpdatedAt = time.Now().UTC()
	e.Version++
	return nil
}

// Seal 封存实验，表示成功结果已确定
func (e *Experiment) Seal() error {
	if e.Status != ExperimentRunning {
		return ErrInvalidStatus
	}
	e.Status = ExperimentRunning
	e.UpdatedAt = time.Now().UTC()
	e.Version++
	return nil
}

// Publish 发布实验
func (e *Experiment) Publish() error {
	if e.Status != ExperimentSealed {
		return ErrInvalidStatus
	}
	e.Status = ExperimentPublished
	e.UpdatedAt = time.Now().UTC()
	e.Version++
	return nil
}

// MarkFailed 标记实验最终失败
func (e *Experiment) MarkFailed() error {
	if e.Status == ExperimentPublished {
		return ErrInvalidStatus
	}
	e.Status = ExperimentFailed
	e.UpdatedAt = time.Now().UTC()
	e.Version++
	return nil
}

// MarshalJSON 自定义序列化以确保时间格式稳定
func (e Experiment) MarshalJSON() ([]byte, error) {
	type Alias Experiment
	return json.Marshal(&struct {
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		*Alias
	}{
		CreatedAt: e.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt: e.UpdatedAt.Format(time.RFC3339Nano),
		Alias:     (*Alias)(&e),
	})
}
