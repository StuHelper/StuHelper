package logger

import (
	"regexp"
	"strings"
)

var (
	emailRegex = regexp.MustCompile(`([a-zA-Z0-9._%+-]+)@([a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`)
	// 中国大陆手机号：1[3-9] 开头共 11 位，添加边界防止匹配更长数字子串
	phoneRegex = regexp.MustCompile(`\b1[3-9]\d{9}\b`)
	// 中国大陆二代身份证号：17 位数字 + 校验位（数字或 X），添加边界防止误匹配
	idCardRegex = regexp.MustCompile(`\b\d{17}[\dXx]\b`)
)

// MaskEmail 脱敏邮箱地址
// example@domain.com -> ex***le@domain.com
func MaskEmail(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***"
	}
	local := parts[0]
	if len(local) <= 2 {
		return "***@" + parts[1]
	}
	return local[:2] + "***" + local[len(local)-2:] + "@" + parts[1]
}

// MaskPhone 脱敏手机号
// 13812345678 -> 138****5678
func MaskPhone(phone string) string {
	if len(phone) != 11 {
		return "***"
	}
	return phone[:3] + "****" + phone[7:]
}

// MaskIDCard 脱敏身份证号
// 110101199001011234 -> 110101********1234
func MaskIDCard(idCard string) string {
	if len(idCard) != 18 {
		return "***"
	}
	return idCard[:6] + "********" + idCard[14:]
}

// MaskToken 脱敏 Token
// eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9... -> eyJh...J9
func MaskToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// MaskIP 脱敏 IP 地址，仅掩码最后一个八位组，保留网络段信息用于问题排查
// 192.168.1.100 -> 192.168.1.***
func MaskIP(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ip // IPv6 或无效格式，不处理
	}
	return parts[0] + "." + parts[1] + "." + parts[2] + ".***"
}

// MaskSensitiveData 自动检测并脱敏字符串中的敏感信息
func MaskSensitiveData(data string) string {
	// 脱敏邮箱
	data = emailRegex.ReplaceAllStringFunc(data, MaskEmail)
	// 脱敏手机号
	data = phoneRegex.ReplaceAllStringFunc(data, MaskPhone)
	// 脱敏身份证
	data = idCardRegex.ReplaceAllStringFunc(data, MaskIDCard)
	return data
}
