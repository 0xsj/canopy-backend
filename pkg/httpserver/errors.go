package httpserver

import (
	"net/http"

	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// KindToStatus maps an error Kind to the appropriate HTTP status code.
func KindToStatus(k canopyerr.Kind) int {
	switch k {
	case canopyerr.KindValidation:
		return http.StatusBadRequest
	case canopyerr.KindAuthentication:
		return http.StatusUnauthorized
	case canopyerr.KindAuthorization:
		return http.StatusForbidden
	case canopyerr.KindNotFound:
		return http.StatusNotFound
	case canopyerr.KindConflict:
		return http.StatusConflict
	case canopyerr.KindRateLimit:
		return http.StatusTooManyRequests
	case canopyerr.KindTimeout:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

// WriteServiceError maps a service error to the appropriate HTTP response.
// 5xx errors are logged but the message is not exposed to the client.
func WriteServiceError(w http.ResponseWriter, log logger.Logger, err error) {
	kind := canopyerr.GetKind(err)
	code := canopyerr.GetCode(err)
	status := KindToStatus(kind)

	codeStr := code.String()
	if codeStr == "" {
		codeStr = kind.String()
	}

	if status >= 500 {
		log.Error("internal error",
			logger.String("code", codeStr),
			logger.Err(err),
		)
		types.WriteError(w, status, codeStr, "internal server error")
		return
	}

	types.WriteError(w, status, codeStr, err.Error())
}
