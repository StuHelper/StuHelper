package singleflightx

import (
	"fmt"

	"golang.org/x/sync/singleflight"
)

// DoValue 执行带返回值的 singleflight 调用，并在调用边界恢复具体类型。
func DoValue[T any](group *singleflight.Group, key string, fn func() (T, error)) (T, error) {
	var zero T

	result, err, _ := group.Do(key, func() (any, error) {
		return fn()
	})
	if err != nil {
		return zero, err
	}

	value, ok := result.(T)
	if !ok {
		return zero, fmt.Errorf("singleflightx: unexpected result type %T", result)
	}
	return value, nil
}

// Do 执行仅关心错误的 singleflight 调用。
func Do(group *singleflight.Group, key string, fn func() error) error {
	_, err := DoValue(group, key, func() (struct{}, error) {
		return struct{}{}, fn()
	})
	return err
}
