package admission

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/sse"
)

// 与 koishi 侧 bots/koishi/packages/shared/src/platform/action-stream-contract.test.ts
// 共享同一份 fixture，双侧字节级锁定 admission action SSE 协议。
const admissionActionStreamContractPath = "../../../api/contracts/admission-action-stream.json"

type admissionActionStreamContract struct {
	CommentOnOpen string            `json:"commentOnOpen"`
	Frames        map[string]string `json:"frames"`
	ActionPayload json.RawMessage   `json:"actionPayload"`
}

func loadAdmissionActionStreamContract(t *testing.T) admissionActionStreamContract {
	t.Helper()

	raw, err := os.ReadFile(admissionActionStreamContractPath)
	if err != nil {
		t.Fatalf("read admission action stream contract: %v", err)
	}
	var contract admissionActionStreamContract
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatalf("parse admission action stream contract: %v", err)
	}
	return contract
}

func newAdmissionStreamRecorder() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func TestAdmissionActionStreamEmitsContractFrames(t *testing.T) {
	gin.SetMode(gin.TestMode)
	contract := loadAdmissionActionStreamContract(t)

	// DisallowUnknownFields 锁定 fixture 字段 ⊆ Go 结构；下方 action 帧字节比对
	// 进一步要求序列化输出与 fixture 一致，即 Go 结构字段 ⊇ fixture 字段。
	decoder := json.NewDecoder(bytes.NewReader(contract.ActionPayload))
	decoder.DisallowUnknownFields()
	var action AdmissionPendingAction
	if err := decoder.Decode(&action); err != nil {
		t.Fatalf("contract actionPayload no longer matches AdmissionPendingAction: %v", err)
	}

	c, w := newAdmissionStreamRecorder()
	if err := sse.WriteComment(c.Writer, admissionStreamOpenComment); err != nil {
		t.Fatalf("write open comment: %v", err)
	}
	if got := w.Body.String(); got != contract.CommentOnOpen {
		t.Fatalf("open comment = %q, want %q", got, contract.CommentOnOpen)
	}

	c, w = newAdmissionStreamRecorder()
	writeAdmissionActionEvent(c, action)
	if got := w.Body.String(); got != contract.Frames[admissionStreamEventAction] {
		t.Fatalf("action frame = %q, want %q", got, contract.Frames[admissionStreamEventAction])
	}

	c, w = newAdmissionStreamRecorder()
	writeAdmissionKeepaliveEvent(c, time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC))
	if got := w.Body.String(); got != contract.Frames[admissionStreamEventKeepalive] {
		t.Fatalf("keepalive frame = %q, want %q", got, contract.Frames[admissionStreamEventKeepalive])
	}

	c, w = newAdmissionStreamRecorder()
	writeAdmissionStreamErrorEvent(c)
	if got := w.Body.String(); got != contract.Frames[admissionStreamEventError] {
		t.Fatalf("error frame = %q, want %q", got, contract.Frames[admissionStreamEventError])
	}

	for name := range contract.Frames {
		switch name {
		case admissionStreamEventAction, admissionStreamEventKeepalive, admissionStreamEventError:
		default:
			t.Fatalf("contract declares event %q the server never emits", name)
		}
	}
}

func TestAdmissionActionStreamContractMatchesOpenAPISchema(t *testing.T) {
	contract := loadAdmissionActionStreamContract(t)

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(contract.ActionPayload, &payload); err != nil {
		t.Fatalf("parse contract actionPayload: %v", err)
	}

	schema := admissionSchemaSection(t, "BotAdmissionPendingAction")
	for key := range payload {
		if !strings.Contains(schema, "\n    "+key+":") {
			t.Fatalf("openapi BotAdmissionPendingAction missing contract field %q", key)
		}
	}
}

// admissionSchemaSection 截取 components/schemas/admission.yaml 中指定顶层 schema 的文本块。
func admissionSchemaSection(t *testing.T, name string) string {
	t.Helper()

	raw, err := os.ReadFile("../../../api/components/schemas/admission.yaml")
	if err != nil {
		t.Fatalf("read admission schema: %v", err)
	}

	lines := strings.Split(string(raw), "\n")
	var section []string
	inSection := false
	for _, line := range lines {
		if line == name+":" {
			inSection = true
			continue
		}
		if inSection {
			if line != "" && !strings.HasPrefix(line, " ") {
				break
			}
			section = append(section, line)
		}
	}
	if !inSection {
		t.Fatalf("schema %q not found in admission.yaml", name)
	}
	return "\n" + strings.Join(section, "\n")
}
