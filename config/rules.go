package config

import (
	"fmt"
	"time"

	"example.com/cssd-instrument-trace-service/domain"
)

// Rules 集中管理领域业务规则，供 service 层读取。
type Rules struct {
	// SterilizationLimits 灭菌参数合格判定下限。
	SterilizationLimits domain.SterilizationLimits
	// PackTypeExpiryDays 各包类型灭菌后的有效天数。
	PackTypeExpiryDays map[domain.PackType]int
	// LostTimeoutHours 发放未回收超过该小时数判定为丢失待查。
	LostTimeoutHours int
	// SweepIntervalMin 过期扫描定时任务的执行间隔（分钟）。
	SweepIntervalMin int
}

// DefaultRules 返回默认业务规则。
func DefaultRules() Rules {
	return Rules{
		SterilizationLimits: domain.SterilizationLimits{
			MinTempC:       134.0,
			MinDurationMin: 4,
			MinPressureKPa: 205.0,
		},
		PackTypeExpiryDays: map[domain.PackType]int{
			domain.TypeSurgical:   7,
			domain.TypeDressing:   14,
			domain.TypeInstrument: 14,
			domain.TypeImplant:    30,
		},
		LostTimeoutHours: 24,
		SweepIntervalMin: 60,
	}
}

// LoadRulesFromEnv 允许通过环境变量覆盖灭菌参数下限等规则。
func LoadRulesFromEnv(r Rules) Rules {
	r.SterilizationLimits = domain.SterilizationLimits{
		MinTempC:       envFloat("STERILIZE_MIN_TEMP", r.SterilizationLimits.MinTempC),
		MinDurationMin: envInt("STERILIZE_MIN_DURATION", r.SterilizationLimits.MinDurationMin),
		MinPressureKPa: envFloat("STERILIZE_MIN_PRESSURE", r.SterilizationLimits.MinPressureKPa),
	}
	r.LostTimeoutHours = envInt("LOST_TIMEOUT_HOURS", r.LostTimeoutHours)
	r.SweepIntervalMin = envInt("SWEEP_INTERVAL_MIN", r.SweepIntervalMin)
	return r
}

// Validate 校验业务规则，防止负数、零值等非法配置导致领域逻辑异常。
func (r Rules) Validate() error {
	if r.SterilizationLimits.MinTempC <= 0 {
		return fmt.Errorf("STERILIZE_MIN_TEMP 必须大于 0")
	}
	if r.SterilizationLimits.MinDurationMin <= 0 {
		return fmt.Errorf("STERILIZE_MIN_DURATION 必须大于 0")
	}
	if r.SterilizationLimits.MinPressureKPa <= 0 {
		return fmt.Errorf("STERILIZE_MIN_PRESSURE 必须大于 0")
	}
	if r.LostTimeoutHours <= 0 {
		return fmt.Errorf("LOST_TIMEOUT_HOURS 必须大于 0")
	}
	if r.SweepIntervalMin <= 0 {
		return fmt.Errorf("SWEEP_INTERVAL_MIN 必须大于 0")
	}
	if len(r.PackTypeExpiryDays) == 0 {
		return fmt.Errorf("PackTypeExpiryDays 不能为空")
	}
	for typ, days := range r.PackTypeExpiryDays {
		if days <= 0 {
			return fmt.Errorf("包类型 %q 的有效期必须大于 0", typ)
		}
	}
	return nil
}

// ExpiryDaysOf 返回指定包类型的有效期天数，未知类型回退为 7 天。
func (r Rules) ExpiryDaysOf(t domain.PackType) int {
	if days, ok := r.PackTypeExpiryDays[t]; ok {
		return days
	}
	return 7
}

// LostTimeout 返回丢失判定时长。
func (r Rules) LostTimeout() time.Duration {
	return time.Duration(r.LostTimeoutHours) * time.Hour
}
