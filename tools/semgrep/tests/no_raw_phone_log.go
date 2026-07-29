package semgreptest

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/phoneutil"
)

func logPhoneValues(log *zap.Logger, phone string) {
	// ruleid: stuhelper.go.raw-phone-log-field
	log.Info("raw phone leaked", zap.String("phone", phone))
	// ruleid: stuhelper.go.raw-phone-log-field
	log.Info("raw mobile leaked", zap.Any("mobile_number", phone))
	// ruleid: stuhelper.go.raw-phone-log-field
	log.Info("formatted phone leaked", zap.String("target_phone", fmt.Sprintf("+86%s", phone)))

	// ok: stuhelper.go.raw-phone-log-field
	log.Info("local helper masked", zap.String("phone", maskPhone(phone)))
	// ok: stuhelper.go.raw-phone-log-field
	log.Info("phoneutil masked", zap.String("phone", phoneutil.Mask(phone)))
	// ok: stuhelper.go.raw-phone-log-field
	log.Info("logger masked", zap.String("phone", logger.MaskPhone(phone)))
	// ok: stuhelper.go.raw-phone-log-field
	log.Info("generic masked", zap.Any("mobile_number", logger.MaskSensitiveData(phone)))
	// ok: stuhelper.go.raw-phone-log-field
	log.Info("non-phone field", zap.String("user_id", phone))
}

func maskPhone(value string) string {
	return value
}
