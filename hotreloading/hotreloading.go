package hotreloading

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

var (
	templatesChanged = make(chan bool)
)

const (
	// Assuming that if template contains <html> tag then the server is sending the full page down
	frameLayoutMarker = "<html>"

	hotReloadURL = "/__hot_reload__"
)

func UseHotRealoadingScriptInjectorMiddleware() func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// Only apply this middleware to HTML responses after the handler
		err := c.Next()

		containsHtml := strings.Contains(string(c.Response().Header.ContentType()), "text/html")

		if err != nil || !containsHtml {
			return err
		}

		log.Println("Injecting WS script")

		// Check if the response body contains the frameLayoutMarker
		body := string(c.Response().Body())
		if strings.Contains(body, frameLayoutMarker) {
			// Inject the WebSocket script into the HTML
			script := fmt.Sprintf(hotreloadingJSTemplate, reconnectingWebSocketsJSBundle, hotReloadURL)
			body = strings.Replace(body, "</body>", script, 1)
			c.SendString(body)
		}

		return nil
	}
}

func SetupWebSocket(app *fiber.App, templateDirs ...string) {
	go startTemplateWatcher(templatesChanged, templateDirs...)

	// Upgrade the conection un a middleware
	app.Use(hotReloadURL, func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// Handle the actaul websocket
	app.Get(hotReloadURL, websocket.New(func(c *websocket.Conn) {
		for {
			select {
			case change := <-templatesChanged:
				if !change {
					continue
				}
				if err := c.WriteMessage(1, []byte("reload")); err != nil {
					log.Println("WebSocket write error:", err)
					return
				}
			}
		}
	}))
}

func startTemplateWatcher(templatesChanged chan bool, templateDirs ...string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					log.Println("Not ok", event)
					return
				}
				log.Println("event:", event)
				templatesChanged <- true
			case err, ok := <-watcher.Errors:
				if !ok {
					log.Println("Not ok error ", err)
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	for _, dir := range templateDirs {
		err = filepath.Walk(dir, walker(watcher))
		if err != nil {
			log.Fatal(err)
		}
	}

	<-done
}

func walker(watcher *fsnotify.Watcher) func(path string, fi os.FileInfo, err error) error {
	return func(path string, fi os.FileInfo, err error) error {
		if fi.Mode().IsDir() {
			log.Print("[HOT_RELOAD] watching: ", path)
			return watcher.Add(path)
		}

		return nil
	}
}
