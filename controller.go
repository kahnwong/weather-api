package main

import "github.com/gofiber/fiber/v2"

func GetTitleController(c *fiber.Ctx) error {
	return c.SendString("Todo")
}
