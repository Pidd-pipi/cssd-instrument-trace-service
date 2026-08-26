package domain

import "errors"

// 领域层统一错误定义，供 store / service / httpapi 各层识别与映射。
var (
	// ErrNotFound 资源不存在。
	ErrNotFound = errors.New("资源不存在")
	// ErrInvalidTransition 非法环节流转（状态机跳步）。
	ErrInvalidTransition = errors.New("非法环节流转，循环必须按序推进")
	// ErrInvalidParam 请求参数不合法。
	ErrInvalidParam = errors.New("参数不合法")
	// ErrDuplicateBarcode 器械包条码已存在。
	ErrDuplicateBarcode = errors.New("器械包条码已存在")
	// ErrSterilizerUnavailable 灭菌器不可用（维护中）。
	ErrSterilizerUnavailable = errors.New("灭菌器当前不可用")
	// ErrIssueBlocked 发放被拦截（未灭菌/已过期/批次参数不合格）。
	ErrIssueBlocked = errors.New("发放被拦截")
	// ErrCollectBlocked 回收被拦截（非使用中或缺少发放记录）。
	ErrCollectBlocked = errors.New("回收被拦截")
	// ErrIssueLost 发放记录已进入「丢失待查」终态，不可再按正常归还回收。
	ErrIssueLost = errors.New("发放记录已丢失待查")
	// ErrConflict 数据状态冲突。
	ErrConflict = errors.New("状态冲突")
)
