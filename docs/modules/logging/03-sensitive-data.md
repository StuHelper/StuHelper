# 敏感信息脱敏

## 必须脱敏的字段

| 字段类型 | 脱敏规则       | 示例                 |
| -------- | -------------- | -------------------- |
| 密码     | 完全隐藏       | `******`             |
| Token    | 保留前后各4位  | `eyJh****gXkI`       |
| 手机号   | 中间4位隐藏    | `138****5678`        |
| 邮箱     | 用户名部分隐藏 | `zh***@example.com`  |
| 身份证   | 中间10位隐藏   | `3201**********1234` |
| 银行卡   | 保留后4位      | `************1234`   |

## 自动脱敏的请求头

以下请求头在日志中自动脱敏：

- `Authorization`
- `Cookie`
- `X-API-Key`
- `X-Auth-Token`

## 脱敏实现代码

```go
// internal/pkg/logger/sensitive.go
package logger

import "strings"

// 敏感字段名（自动脱敏）
var sensitiveFields = map[string]bool{
    "password": true, "token": true, "secret": true,
    "api_key": true, "authorization": true, "cookie": true,
}

// MaskPhone 手机号脱敏
func MaskPhone(phone string) string {
    if len(phone) != 11 {
        return "***********"
    }
    return phone[:3] + "****" + phone[7:]
}

// MaskEmail 邮箱脱敏
func MaskEmail(email string) string {
    parts := strings.Split(email, "@")
    if len(parts) != 2 {
        return "****@****.***"
    }
    name := parts[0]
    if len(name) > 2 {
        name = name[:2] + strings.Repeat("*", len(name)-2)
    }
    return name + "@" + parts[1]
}

// MaskToken Token 脱敏
func MaskToken(token string) string {
    if len(token) <= 8 {
        return "****"
    }
    return token[:4] + "****" + token[len(token)-4:]
}

// MaskIDCard 身份证脱敏
func MaskIDCard(idCard string) string {
    if len(idCard) < 8 {
        return "******************"
    }
    return idCard[:4] + strings.Repeat("*", len(idCard)-8) + idCard[len(idCard)-4:]
}
```
