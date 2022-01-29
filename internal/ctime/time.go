package ctime

import (
	"context"
	"time"
)

// 用于将具体的时间字符串解析成duration类型
// 例如 1s, 500ms
type Duration time.Duration

// 反序列化方法
func (d *Duration) UnmarshalText(text []byte) error {
	tmp, err := time.ParseDuration(string(text))
	if err == nil {
		*d = Duration(tmp)
	}
	return err
}

// 将当前时间与context超时时间比较，并缩短至最小值
// 若当前context不存在超时时间，则使用duration作为超时时间，并返回包含超时时间的新context
func (d Duration) Shrink(c context.Context) (Duration, context.Context, context.CancelFunc) {
	if deadline, ok := c.Deadline(); ok {
		if timeout := time.Until(deadline); timeout < time.Duration(d) {
			return Duration(timeout), c, func() {}
		}
	}
	ctx, cancel := context.WithTimeout(c, time.Duration(d))
	return d, ctx, cancel
}
