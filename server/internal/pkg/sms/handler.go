package sms

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// RegisterInternalHandler 注册内部 SMS 转发端点。
// Casdoor Custom HTTP SMS Provider 通过 HTTP 调用此端点发送验证码短信。
// 端点使用 InternalKey 做简单鉴权，生产环境应配合网络隔离。
func (s *Service) RegisterInternalHandler(mux *http.ServeMux) {
	mux.HandleFunc("POST /internal/sms/send", s.handleSend)
}

func (s *Service) handleSend(w http.ResponseWriter, r *http.Request) {
	// 鉴权：Bearer token = InternalKey（fail-closed：未配置时拒绝所有请求）
	if s.cfg.InternalKey == "" {
		s.logger.Error("SMS internal key not configured, rejecting request")
		writeJSON(w, http.StatusForbidden, SendResponse{Error: "service not configured"})
		return
	}
	token := r.Header.Get("Authorization")
	expected := "Bearer " + s.cfg.InternalKey
	if subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
		writeJSON(w, http.StatusUnauthorized, SendResponse{Error: "unauthorized"})
		return
	}

	var req SendRequest
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, SendResponse{Error: "invalid request body"})
		return
	}

	if req.Phone == "" || req.Content == "" {
		writeJSON(w, http.StatusBadRequest, SendResponse{Error: "phone and content are required"})
		return
	}

	if err := s.Send(r.Context(), req.Phone, req.Content); err != nil {
		s.logger.Error("SMS send failed", zap.String("phone", maskPhone(req.Phone)), zap.Error(err))
		writeJSON(w, http.StatusBadGateway, SendResponse{Error: "failed to send SMS"})
		return
	}

	writeJSON(w, http.StatusOK, SendResponse{Success: true})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
