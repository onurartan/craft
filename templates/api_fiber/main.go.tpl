package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	app := fiber.New(fiber.Config{
		AppName: "{{ .ProjectName }}",
	})

	app.Use(logger.New())

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Welcome to {{ .ProjectName }} API!",
			"status":  "success",
		})
	})

	log.Println("Server starting on port 3000...")
	log.Fatal(app.Listen(":3000"))
}
