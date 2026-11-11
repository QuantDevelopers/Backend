package controller

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

type SingleMessageResponse struct {
	Status  int         `json:"statusCode"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type MultipleMessageResponse struct {
	Status   int                          `json:"statusCode"`
	Messages map[string]map[string]string `json:"messages"`
	Data     interface{}                  `json:"data"`
}

func Response[T string | map[string]map[string]string](ctx echo.Context, statusCode int, message T, data interface{}) error {
	statusText := http.StatusText(statusCode)
	if statusText == "" {
		panic(fmt.Errorf("invalid http status code: %d", statusCode))
	}

	switch msg := any(message).(type) {
	case string:
		if msg == "" {
			msg = statusText
		}
		return ctx.JSON(statusCode, SingleMessageResponse{
			Status:  statusCode,
			Message: msg,
			Data:    data,
		})
	case map[string]map[string]string:
		return ctx.JSON(statusCode, MultipleMessageResponse{
			Status:   statusCode,
			Messages: msg,
			Data:     data,
		})
	default:
		return nil
	}
}
