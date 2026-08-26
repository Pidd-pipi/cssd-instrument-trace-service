// Package service 实现业务用例编排，连接 store 与 httpapi。
package service

import "errors"

// BlockedError 携带拦截原因的领域错误，供 httpapi 映射为 422 并返回具体原因。
type BlockedError struct {
	Kind   error
	Reason string
}

// Error 实现 error 接口。
func (e *BlockedError) Error() string { return e.Reason }

// Unwrap 返回底层哨兵错误，便于 errors.Is 判断。
func (e *BlockedError) Unwrap() error { return e.Kind }

// IsBlocked 判断错误是否为业务拦截类错误。
func IsBlocked(err error) bool {
	var be *BlockedError
	return errors.As(err, &be)
}
