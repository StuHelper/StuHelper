package sms

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

const (
	internalKeyQueryParam       = "internal_key"
	casdoorSMSPhoneFormField    = "phoneNumber"
	casdoorSMSContentFormField  = "content"
	maxInternalSendRequestBytes = 4096
)

// RegisterInternalHandler 注册内部 SMS 转发端点。
// Casdoor Custom HTTP SMS Provider 通过 HTTP 调用此端点发送验证码短信。
// 端点使用 InternalKey 做简单鉴权，生产环境应配合网络隔离。
func (s *Service) RegisterInternalHandler(mux *http.ServeMux) {
	mux.HandleFunc("POST /internal/sms/send", s.handleSend)
}

func (s *Service) handleSend(w http.ResponseWriter, r *http.Request) {
	// 鉴权：Casdoor form callback 用 internal_key query；内部诊断调用可用 Bearer token。
	if s.cfg.InternalKey == "" {
		s.logger.Error("SMS internal key not configured, rejecting request")
		writeJSON(w, http.StatusForbidden, SendResponse{Error: "service not configured"})
		return
	}
	if !s.authorizeInternalRequest(r) {
		writeJSON(w, http.StatusUnauthorized, SendResponse{Error: "unauthorized"})
		return
	}

	req, err := decodeInternalSendRequest(w, r)
	if err != nil {
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

func (s *Service) authorizeInternalRequest(r *http.Request) bool {
	expected := "Bearer " + s.cfg.InternalKey
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte(expected)) == 1 {
		return true
	}
	return subtle.ConstantTimeCompare([]byte(r.URL.Query().Get(internalKeyQueryParam)), []byte(s.cfg.InternalKey)) == 1
}

func decodeInternalSendRequest(w http.ResponseWriter, r *http.Request) (SendRequest, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxInternalSendRequestBytes)
	if isFormURLEncoded(r.Header.Get("Content-Type")) {
		return decodeCasdoorSMSForm(r)
	}
	return decodeJSONSendRequest(r)
}

func decodeJSONSendRequest(r *http.Request) (SendRequest, error) {
	var req SendRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return SendRequest{}, err
	}
	var extra struct{}
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return SendRequest{}, errors.New("request body must contain a single JSON object")
		}
		return SendRequest{}, err
	}
	return req, nil
}

func isFormURLEncoded(contentType string) bool {
	return strings.HasPrefix(strings.ToLower(contentType), "application/x-www-form-urlencoded")
}

func decodeCasdoorSMSForm(r *http.Request) (SendRequest, error) {
	if err := r.ParseForm(); err != nil {
		return SendRequest{}, err
	}
	req := SendRequest{
		Phone:   r.FormValue(casdoorSMSPhoneFormField),
		Content: r.FormValue(casdoorSMSContentFormField),
	}
	if req.Phone == "" || req.Content == "" {
		return SendRequest{}, errors.New("casdoor sms form requires phoneNumber and content")
	}
	return req, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
