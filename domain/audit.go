package domain

import "time"

// AuditLog 操作审计日志：环节流转、灭菌判定、发放回收等关键操作全部留痕。
type AuditLog struct {
	ID         string         `json:"id"`
	Action     string         `json:"action"` // 例如 pack.register / sterilization.complete
	Operator   string         `json:"operator"`
	TargetType string         `json:"targetType"`
	TargetID   string         `json:"targetId"`
	Detail     map[string]any `json:"detail,omitempty"`
	IP         string         `json:"ip,omitempty"`
	CreatedAt  time.Time      `json:"createdAt"`
}

// 审计动作常量，前端 audit 日志页与后端共用语义。
const (
	ActionPackRegister          = "pack.register"
	ActionPackCycle             = "pack.cycle"
	ActionSterilizationCreate   = "sterilization.create"
	ActionSterilizationComplete = "sterilization.complete"
	ActionPackIssue             = "pack.issue"
	ActionPackCollect           = "pack.collect"
	ActionExpirySweep           = "expiry.sweep"
	ActionSterilizerCreate      = "sterilizer.create"
	ActionHTTPRequest           = "http.request"
)

// NewAuditLog 构建审计日志。
func NewAuditLog(action, operator, targetType, targetID, ip string, detail map[string]any) *AuditLog {
	return &AuditLog{
		ID:         NewID("audit"),
		Action:     action,
		Operator:   operator,
		TargetType: targetType,
		TargetID:   targetID,
		Detail:     detail,
		IP:         ip,
		CreatedAt:  time.Now(),
	}
}
