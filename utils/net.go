package utils

import "github.com/gofiber/fiber/v2"

func IPAddress(c *fiber.Ctx) string {
	IPAddress := c.Get("CF-Connecting-IP")
	if IPAddress == "" {
		IPAddress = c.Get("X-Real-Ip")
	}
	if IPAddress == "" {
		IPAddress = c.Get("X-Forwarded-For")
	}
	if IPAddress == "" {
		IPAddress = c.IP()
	}
	return IPAddress
}
