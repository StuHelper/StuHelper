package auth

import "time"

func (h *Handler) accessTokenTTL() time.Duration {
	if h == nil {
		return 0
	}
	if h.tokenService == nil {
		return time.Duration(h.tokenConfig.AccessTokenTTL) * time.Second
	}
	return h.tokenService.GetAccessTokenTTL()
}

func (h *Handler) accessTokenTTLSeconds() int {
	return int(h.accessTokenTTL() / time.Second)
}
