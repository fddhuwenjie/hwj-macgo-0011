package util

import (
	"errors"
	"fmt"
)

// WrapError 包装错误并添加上下文
func WrapError(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", msg, err)
}

// WrapErrorf 格式化包装错误
func WrapErrorf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}

// Is 判断错误是否匹配目标
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// JoinErrors 合并多个错误
func JoinErrors(errs ...error) error {
	return errors.Join(errs...)
}
