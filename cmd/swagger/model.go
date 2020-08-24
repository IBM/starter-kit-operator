package swagger

import (
	"net/http"

	"github.com/ibm/starter-kit-operator/cmd/swagger/api"
)

// CodeResponse is a generic success response
type CodeResponse struct {
	Code    int    `json:"code"`
	Message string `json:"msg,omitempty"`
}

// BooleanResponse is a generic success response containing code and a boolean value
type BooleanResponse struct {
	Code    int    `json:"code"`
	Data    bool   `json:"data"`
	Message string `json:"msg,omitempty"`
}

// CodeOK returns HTTP Status OK (200)
func CodeOK() CodeResponse {
	return CodeResponse{Code: http.StatusOK}
}

// NewBoolResponse returns new Boolean response
func NewBoolResponse(b bool) BooleanResponse {
	return BooleanResponse{Code: http.StatusOK, Data: b}
}

// Success response
// swagger:response ok
type swaggScsResp struct {
	// in:body
	Body struct {
		// HTTP Status Code 200
		Code int `json:"code"`
	}
}

// Boolean response
// swagger:response bool
type swaggBoolResp struct {
	// in:body
	Body struct {
		// HTTP Status Code 200
		Code int `json:"code"`
		// Boolean true/false
		Data bool `json:"data"`
	}
}

// Error Bad Request
// swagger:response badReq
type swaggErrBadReq struct {
	// in:body
	Body struct {
		// HTTP status code 400 - Status Bad Request
		Code int `json:"code"`
		// Detailed error message
		Message string `json:"message"`
	}
}

// Error Forbidden
// swagger:response forbidden
type swaggErrForbidden struct {
	// in:body
	Body struct {
		// HTTP status code 403 - Forbidden
		Code int `json:"code"`
		// Detailed error message
		Message string `json:"message"`
	}
}

// Error Not Found
// swagger:response notFound
type swaggErrNotFound struct {
	// in:body
	Body struct {
		// HTTP status code 404 - Not Found
		Code int `json:"code"`
		// Detailed error message
		Message string `json:"message"`
	}
}

// Error Conflict
// swagger:response conflict
type swaggErrConflict struct {
	// in:body
	Body struct {
		// HTTP status code 409 - Conflict
		Code int `json:"code"`
		// Detailed error message
		Message string `json:"message"`
	}
}

// Error Interval Server
// swagger:response internal
type swaggErrInternal struct {
	// in:body
	Body struct {
		// HTTP status code 500 - Internal server error
		Code int `json:"code"`
		// Detailed error message
		Message string `json:"message"`
	}
}

// Repository model request
// swagger:parameters repoReq
type swaggRepoReq struct {
	// in:body
	Body model.Repository
}

// Request containing string
// swagger:parameters createRepoReq
type swaggerCreateRepoReq struct {
	// in:body
	Body api.CreateRepoReq
}

// HTTP status code 200 and repository model in data
// swagger:response repoResp
type swaggRepoResp struct {
	// in:body
	Body struct {
		// HTTP status code 200 - Status OK
		Code int `json:"code"`
		// Repository model
		Data model.Repository `json:"data"`
	}
}

// HTTP status code 200 and an array of starter kit models in data
// swagger:response reposResp
type swaggStarterKitsResp struct {
	// in:body
	Body struct {
		// HTTP status code 200 - Status OK
		Code int `json:"code"`
		// Array of repository models
		Data []model.StarterKit `json:"data"`
	}
}
