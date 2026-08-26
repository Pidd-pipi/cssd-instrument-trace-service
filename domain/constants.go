// Package domain 定义领域实体、枚举与状态机规则。
package domain

// PackStage 器械包环节枚举。
// 前后端共享枚举定义：后端见 domain/constants.go、domain/pack_stage.go，
// 前端见 web/constants.js（PackStage 与 stageLabels 保持一致）。
type PackStage string

const (
	// StageToCollect 待回收：器械包使用完毕后等待回收。
	StageToCollect PackStage = "to_collect"
	// StageCollected 已回收：器械包已被回收至消毒供应中心。
	StageCollected PackStage = "collected"
	// StageWashing 清洗中：器械包处于清洗/消毒流程。
	StageWashing PackStage = "washing"
	// StageWashed 已清洗：清洗消毒完成，等待灭菌。
	StageWashed PackStage = "washed"
	// StageSterilizing 灭菌中：器械包已装载进入灭菌批次。
	StageSterilizing PackStage = "sterilizing"
	// StageSterilized 已灭菌：灭菌参数合格，处于无菌存放期。
	StageSterilized PackStage = "sterilized"
	// StageIssued 已发放：器械包已发放至使用科室。
	StageIssued PackStage = "issued"
	// StageInUse 使用中：器械包正在手术间使用。
	StageInUse PackStage = "in_use"
	// StageExpired 已过期：无菌有效期已过，禁止发放，需重新清洗灭菌。
	StageExpired PackStage = "expired"
)

// SterilizeResult 灭菌参数判定结果枚举。
type SterilizeResult string

const (
	// ResultPass 参数合格。
	ResultPass SterilizeResult = "pass"
	// ResultFail 参数不合格，批次内器械包全部拦截不得发放。
	ResultFail SterilizeResult = "fail"
)

// PackType 器械包包类型枚举。
type PackType string

const (
	// TypeSurgical 手术器械包。
	TypeSurgical PackType = "surgical"
	// TypeDressing 敷料包。
	TypeDressing PackType = "dressing"
	// TypeInstrument 器械包。
	TypeInstrument PackType = "instrument"
	// TypeImplant 植入物包。
	TypeImplant PackType = "implant"
)

// IssueStatus 发放记录状态枚举。
type IssueStatus string

const (
	// IssueOpen 已发放、尚未回收。
	IssueOpen IssueStatus = "issued"
	// IssueReturned 已回收闭环。
	IssueReturned IssueStatus = "returned"
	// IssueLost 丢失待查：发放超过时限未回收。
	IssueLost IssueStatus = "lost"
)

// SterilizerStatus 灭菌器状态枚举。
type SterilizerStatus string

const (
	// SterilizerAvailable 可用。
	SterilizerAvailable SterilizerStatus = "available"
	// SterilizerMaintenance 维护中，不可用于创建批次。
	SterilizerMaintenance SterilizerStatus = "maintenance"
)

// BatchStatus 灭菌批次状态枚举。
type BatchStatus string

const (
	// BatchPending 批次已创建、参数判定未完成。
	BatchPending BatchStatus = "pending"
	// BatchCompleted 批次已完结（参数判定完成）。
	BatchCompleted BatchStatus = "completed"
)
