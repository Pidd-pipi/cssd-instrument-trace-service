package domain

import "fmt"

// stageTransitions 器械包循环状态机唯一合法的相邻迁移表。
// 循环顺序：待回收 → 已回收 → 清洗中 → 已清洗 → 灭菌中 → 已灭菌 → 已发放 → 使用中 → 回到待回收。
// 过期包通过「重新清洗」回到清洗流程。
var stageTransitions = map[PackStage]PackStage{
	StageToCollect:   StageCollected,
	StageCollected:   StageWashing,
	StageWashing:     StageWashed,
	StageWashed:      StageSterilizing,
	StageSterilizing: StageSterilized,
	StageSterilized:  StageIssued,
	StageIssued:      StageInUse,
	StageInUse:       StageToCollect,
	StageExpired:     StageWashing,
}

// manualCycleTransitions 允许通过通用「环节流转」接口手动推进的迁移。
// 以下迁移必须走专用接口，禁止手动流转：
//   - 已清洗→灭菌中 / 灭菌中→已灭菌：由灭菌批次创建/完结驱动；
//   - 已灭菌→已发放：必须经发放接口完成三项校验；
//   - 使用中→待回收：必须经回收接口闭环发放记录。
// 注意：StageExpired→StageWashing 允许手动流转，过期包需重新清洗灭菌。
var manualCycleTransitions = map[PackStage]PackStage{
	StageToCollect: StageCollected, // 待回收→已回收
	StageCollected: StageWashing,   // 已回收→清洗中
	StageWashing:   StageWashed,    // 清洗中→已清洗
	StageIssued:    StageInUse,     // 已发放→使用中
	StageExpired:   StageWashing,   // 已过期→清洗中（重新处理）
}

// ValidStages 返回全部合法环节枚举值。
func ValidStages() []PackStage {
	return []PackStage{
		StageToCollect, StageCollected, StageWashing, StageWashed,
		StageSterilizing, StageSterilized, StageIssued, StageInUse, StageExpired,
	}
}

// IsValidStage 判断字符串是否为合法环节枚举。
func IsValidStage(s PackStage) bool {
	for _, st := range ValidStages() {
		if st == s {
			return true
		}
	}
	return false
}

// NextStage 返回状态机中指定环节的下一环节。
func NextStage(s PackStage) (PackStage, bool) {
	next, ok := stageTransitions[s]
	return next, ok
}

// ValidateTransition 校验 from→to 是否为状态机允许的相邻迁移。
// 过期包（expired）允许重新进入清洗流程（expired→washing）。
func ValidateTransition(from, to PackStage) error {
	expected, ok := stageTransitions[from]
	if !ok {
		return fmt.Errorf("%w: 当前环节 %s 无合法后继", ErrInvalidTransition, from)
	}
	if expected != to {
		return fmt.Errorf("%w: 环节 %s 的下一环节应为 %s，实际请求 %s", ErrInvalidTransition, from, expected, to)
	}
	return nil
}

// IsManualCycle 判断 from→to 是否允许通过通用环节流转接口手动推进。
// 已清洗→灭菌中、灭菌中→已灭菌、已灭菌→已发放、使用中→待回收
// 必须走专用接口（灭菌批次创建/完结、发放接口、回收接口），禁止手动流转。
func IsManualCycle(from, to PackStage) bool {
	expected, ok := manualCycleTransitions[from]
	return ok && expected == to
}
