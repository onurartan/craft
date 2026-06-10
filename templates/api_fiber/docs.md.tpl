# {{ .ProjectName }} - API (Fiber) Documentation

Welcome to your new blazing fast REST API, bootstrapped with Craft! 
This project is pre-configured with industry-standard tools to get you building immediately.

## 📦 Default Packages & Tech Stack

This template comes with carefully selected default packages to ensure high performance and DX (Developer Experience):

- **`github.com/gofiber/fiber/v2`**: The core web framework. Inspired by Express.js, Fiber is built on top of Fasthttp, the fastest HTTP engine for Go. It is incredibly fast, memory efficient, and heavily focused on rapid development.
- **`github.com/gofiber/fiber/v2/middleware/logger`**: Built-in Fiber middleware that logs HTTP requests and errors, crucial for debugging and monitoring your API traffic.

## 🚀 Getting Started

### 1. Development Mode (Hot-Reload)
Instead of manually restarting the server every time you make a code change, use Craft's magic:
```bash
craft dev
```
- Craft will watch your `.go` files. 
- Every time you hit `CTRL+S`, the server will restart in milliseconds.
- If you import a new package (e.g., `import "github.com/google/uuid"`), Craft's **Auto-Install** feature will automatically download it without crashing!

### 2. Adding a New Route
Open `main.go`. Inside the `main()` function, you can start building your API endpoints right away:
```go
// Add this below app.Get("/", ...)
app.Post("/users", func(c *fiber.Ctx) error {
	// Parse JSON request
	var user struct { Name string }
	if err := c.BodyParser(&user); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}
	
	// Return JSON response
	return c.JSON(fiber.Map{
		"message": "User " + user.Name + " created successfully!",
	})
})
```

### 3. Adding New Packages
Use Craft's shorthand syntax to add new libraries effortlessly:
```bash
craft add gh:joho/godotenv  # Translates to github.com/joho/godotenv
```

### 4. Production Build
Ready to deploy? Compile your code into a single, highly optimized binary:
```bash
craft build
```
Your compiled `.exe` (or binary) will be sitting in the `bin/` directory, ready to be executed on your server or packaged into a Docker container.

## 📂 Project Structure
- `main.go`: Application entry point and routing setup.
- `.craft.yaml`: Your project's core configuration. Use the `commands` section to create custom tasks (like `craft setup` or `craft db-migrate`), and the `minify` section if you decide to serve frontend assets.

Happy coding! 🔥
