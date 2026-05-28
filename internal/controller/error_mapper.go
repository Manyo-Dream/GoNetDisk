package controller

import (
	"errors"
	"net/http"

	"GoNetDisk/internal/util"
)

func statusFromErr(err error) int {
	switch {
	case errors.Is(err, util.ErrBadRequest):
		return http.StatusBadRequest
	case errors.Is(err, util.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, util.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, util.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, util.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, util.ErrNotImplemented):
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}
