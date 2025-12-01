package validator

import (
	"errors"
	"regexp"
	"unicode"
)

// 用户名验证规则：
// - 长度 >= 2
// - 仅大小写字母和数字
// - 不能有特殊字符
func ValidateUsername(username string) error {
	if len(username) < 2 {
		return errors.New("用户名长度不能少于2个字符")
	}
	if len(username) > 32 {
		return errors.New("用户名长度不能超过32个字符")
	}

	// 只允许字母和数字
	matched, _ := regexp.MatchString("^[a-zA-Z0-9]+$", username)
	if !matched {
		return errors.New("用户名只能包含字母和数字")
	}

	return nil
}

// 密码验证规则：
// - 长度 >= 6
// - 不能有连续3个及以上相同字符
// - 不能是连续字符 (abc, 123, cba, 321)
func ValidatePassword(password string) error {
	if len(password) < 6 {
		return errors.New("密码长度不能少于6个字符")
	}
	if len(password) > 64 {
		return errors.New("密码长度不能超过64个字符")
	}

	// 检查连续3个及以上相同字符
	if hasRepeatingChars(password, 3) {
		return errors.New("密码不能包含3个及以上连续相同的字符")
	}

	// 检查连续递增/递减字符
	if hasSequentialChars(password, 3) {
		return errors.New("密码不能包含3个及以上连续的字符(如abc、123)")
	}

	return nil
}

// hasRepeatingChars 检查是否有连续重复字符
func hasRepeatingChars(s string, count int) bool {
	if len(s) < count {
		return false
	}

	runes := []rune(s)
	repeatCount := 1

	for i := 1; i < len(runes); i++ {
		if runes[i] == runes[i-1] {
			repeatCount++
			if repeatCount >= count {
				return true
			}
		} else {
			repeatCount = 1
		}
	}

	return false
}

// hasSequentialChars 检查是否有连续递增/递减字符
func hasSequentialChars(s string, count int) bool {
	if len(s) < count {
		return false
	}

	runes := []rune(s)

	// 检查递增序列
	incCount := 1
	decCount := 1

	for i := 1; i < len(runes); i++ {
		// 递增检查 (a->b, 1->2)
		if runes[i] == runes[i-1]+1 {
			incCount++
			if incCount >= count {
				return true
			}
		} else {
			incCount = 1
		}

		// 递减检查 (c->b, 3->2)
		if runes[i] == runes[i-1]-1 {
			decCount++
			if decCount >= count {
				return true
			}
		} else {
			decCount = 1
		}
	}

	return false
}

// ValidateEmail 验证邮箱格式（可选字段）
func ValidateEmail(email string) error {
	if email == "" {
		return nil // 邮箱是可选的
	}

	// 简单的邮箱格式验证
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, email)
	if !matched {
		return errors.New("邮箱格式不正确")
	}

	return nil
}

// IsAlphanumeric 检查字符串是否只包含字母和数字
func IsAlphanumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
