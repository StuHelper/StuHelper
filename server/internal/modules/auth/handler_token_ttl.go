package auth

import "time"

func (h *Handler) currentAccessTokenTTL() time.Duration {
	if h.tokenService != nil {
		ttl := h.tokenService.GetAccessTokenTTL()
		if ttl > 0 {
			return ttl
		}
	}
	return time.Duration(h.tokenConfig.AccessTokenTTL) * time.Second
}

func (h *Handler) currentAccessTokenTTLSeconds() int {
	return int(h.currentAccessTokenTTL() / time.Second)
}
