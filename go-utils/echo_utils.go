package goutils

// using echo web frame work for communication layer integration btw clinet and server
import "github.com/labstack/echo/v4"

func NewEchoServer() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	return e;
}