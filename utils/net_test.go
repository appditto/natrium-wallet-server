package utils

import (
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

func TestGetIPAddressFromHeader(t *testing.T) {
	ip := "123.45.67.89"

	// 4 methods of getting IP Address, CF-Connecting-IP preferred, X-Real-Ip, then X-Forwarded-For, then RemoteAddr

	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	c.Request().Header.Set("CF-Connecting-IP", ip)
	c.Request().Header.Set("X-Real-Ip", "not-the-ip")
	c.Request().Header.Set("X-Forwarded-For", "not-the-ip")
	AssertEqual(t, c.Get("CF-Connecting-IP"), ip)
	AssertEqual(t, ip, IPAddress(c))
	app.ReleaseCtx(c)

	c = app.AcquireCtx(&fasthttp.RequestCtx{})
	c.Request().Header.Set("X-Real-Ip", ip)
	c.Request().Header.Set("X-Forwarded-For", "not-the-ip")
	AssertEqual(t, ip, IPAddress(c))
	app.ReleaseCtx(c)

	c = app.AcquireCtx(&fasthttp.RequestCtx{})
	c.Request().Header.Set("X-Forwarded-For", ip)
	AssertEqual(t, ip, IPAddress(c))
	app.ReleaseCtx(c)
}
