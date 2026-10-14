// Package httpx holds the JSON envelope and error helpers shared by all features.
//
// Success:  {"data": ..., "meta": {...}}
// Failure:  {"error": {"code": "...", "message": "...", "details": ..., "request_id": "..."}}
package httpx

import (
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// RequestIDKey is the gin context key under which the request ID is stored.
const RequestIDKey = "request_id"

type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Details   any    `json:"details,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

type Meta struct {
	Page    int   `json:"page"`
	PerPage int   `json:"per_page"`
	Total   int64 `json:"total"`
}

// JSON writes {"data": data} with the given status.
func JSON(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{"data": data})
}

// Paginated writes {"data": data, "meta": meta}.
func Paginated(c *gin.Context, data any, meta Meta) {
	c.JSON(http.StatusOK, gin.H{"data": data, "meta": meta})
}

func body(c *gin.Context, code, message string, details any) ErrorBody {
	return ErrorBody{Error: ErrorDetail{
		Code:      code,
		Message:   message,
		Details:   details,
		RequestID: c.GetString(RequestIDKey),
	}}
}

// Error writes an error envelope.
func Error(c *gin.Context, status int, code, message string) {
	c.JSON(status, body(c, code, message, nil))
}

// AbortError writes an error envelope and aborts the handler chain (middleware use).
func AbortError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, body(c, code, message, nil))
}

// BindError converts a gin binding error into a 400 (malformed) or 422 (validation) response.
func BindError(c *gin.Context, err error) {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		fields := make(map[string]string, len(ve))
		for _, fe := range ve {
			fields[fe.Field()] = "failed '" + fe.Tag() + "' validation"
		}
		c.JSON(http.StatusUnprocessableEntity, body(c, "validation_error", "Request validation failed", fields))
		return
	}
	Error(c, http.StatusBadRequest, "invalid_request", "Malformed request")
}

// RegisterJSONTagNames makes validation errors report json/form field names
// (e.g. "email") instead of Go struct field names (e.g. "Email"). Call once at startup.
func RegisterJSONTagNames() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		for _, tag := range []string{"json", "form"} {
			name := strings.SplitN(f.Tag.Get(tag), ",", 2)[0]
			if name == "-" {
				return ""
			}
			if name != "" {
				return name
			}
		}
		return f.Name
	})
}
