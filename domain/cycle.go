package domain

import "time"

// CycleRecord 环节记录：器械包每个环节的操作留痕，构成完整追溯链。
type CycleRecord struct {
	ID        string         `json:"id"`
	PackID    string         `json:"packId"`
	Barcode   string         `json:"barcode"`
	FromStage PackStage      `json:"fromStage"`
	Stage     PackStage      `json:"stage"`
	Operator  string         `json:"operator"`
	DeviceID  string         `json:"deviceId,omitempty"`
	Note      string         `json:"note,omitempty"`
	Params    map[string]any `json:"params,omitempty"` // 参数快照（温度/时长/压力/科室等）
	CreatedAt time.Time      `json:"createdAt"`
}

// Copy 返回环节记录的深拷贝，避免调用方意外修改仓储内对象。
func (c *CycleRecord) Copy() *CycleRecord {
	if c == nil {
		return nil
	}
	cp := *c
	if c.Params != nil {
		cp.Params = copyParams(c.Params)
	}
	return &cp
}

// copyParams 浅拷贝一层参数 map，保证变更 fn 拿到的是副本而非仓储内引用。
func copyParams(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// NewCycleRecord 构建一条环节记录。
// from 为迁移前环节（登记时为空字符串），to 为迁移后环节。
func NewCycleRecord(pack *InstrumentPack, from, to PackStage, operator, deviceID, note string, params map[string]any) *CycleRecord {
	return &CycleRecord{
		ID:        NewID("cycle"),
		PackID:    pack.ID,
		Barcode:   pack.Barcode,
		FromStage: from,
		Stage:     to,
		Operator:  operator,
		DeviceID:  deviceID,
		Note:      note,
		Params:    params,
		CreatedAt: time.Now(),
	}
}
