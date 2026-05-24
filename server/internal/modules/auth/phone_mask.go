package auth

// maskPhone 隐藏手机号中间四位，用于审计与管理日志中的账号标识。
func maskPhone(phone string) string {
	if len(phone) < 7 {
		return "***"
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}
