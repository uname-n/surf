package browser

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/dop251/goja"
)

// JSEngine handles JavaScript execution for the browser
type JSEngine struct {
	vm      *goja.Runtime
	browser *Browser
	dom     *goquery.Document
}

// NewJSEngine creates a new JavaScript engine
func NewJSEngine(browser *Browser) *JSEngine {
	return &JSEngine{
		vm:      goja.New(),
		browser: browser,
	}
}

// Execute runs JavaScript code in the current page context
func (js *JSEngine) Execute(doc *goquery.Document) error {
	js.dom = doc

	// Extract all script tags and execute them
	inlineScripts := []string{}
	externalScripts := []string{}

	// First collect scripts
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists {
			// External script
			resolvedSrc, err := js.browser.ResolveStringUrl(src)
			if err == nil {
				externalScripts = append(externalScripts, resolvedSrc)
			}
		} else {
			// Inline script
			if code := s.Text(); strings.TrimSpace(code) != "" {
				inlineScripts = append(inlineScripts, code)
			}
		}
	})

	// Set up the document object
	err := js.setupDOM()
	if err != nil {
		return err
	}

	// Fetch and execute external scripts
	for _, src := range externalScripts {
		code, err := js.fetchExternalScript(src)
		if err != nil {
			fmt.Printf("Error fetching script %s: %v\n", src, err)
			continue
		}

		_, err = js.vm.RunString(code)
		if err != nil {
			fmt.Printf("JavaScript error in %s: %v\n", src, err)
		}
	}

	// Execute inline scripts
	for _, script := range inlineScripts {
		_, err := js.vm.RunString(script)
		if err != nil {
			// We don't fail on JS errors, just log them
			fmt.Printf("JavaScript error: %v\n", err)
		}
	}

	return nil
}

// fetchExternalScript downloads an external JavaScript file
func (js *JSEngine) fetchExternalScript(url string) (string, error) {
	// Create a new HTTP client to fetch the script
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Use the same user agent as the browser
	req.Header.Set("User-Agent", js.browser.userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Failed to fetch script, status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// setupDOM creates a minimal document object for scripts to work with
func (js *JSEngine) setupDOM() error {
	// Create a basic window object
	windowObj := js.vm.NewObject()

	// Create a document object
	documentObj := js.vm.NewObject()

	// Setup document methods
	err := documentObj.Set("getElementById", js.vm.ToValue(func(id string) goja.Value {
		// Find element by ID in the DOM
		selection := js.dom.Find("#" + id)
		if selection.Length() == 0 {
			return goja.Null()
		}
		return js.elementToJSValue(selection)
	}))
	if err != nil {
		return err
	}

	err = documentObj.Set("getElementsByTagName", js.vm.ToValue(func(tagName string) goja.Value {
		// Find elements by tag name
		selection := js.dom.Find(tagName)
		return js.selectionToJSArray(selection)
	}))
	if err != nil {
		return err
	}

	err = documentObj.Set("querySelector", js.vm.ToValue(func(selector string) goja.Value {
		// Find first element matching selector
		selection := js.dom.Find(selector)
		if selection.Length() == 0 {
			return goja.Null()
		}
		return js.elementToJSValue(selection.First())
	}))
	if err != nil {
		return err
	}

	err = documentObj.Set("querySelectorAll", js.vm.ToValue(func(selector string) goja.Value {
		// Find all elements matching selector
		selection := js.dom.Find(selector)
		return js.selectionToJSArray(selection)
	}))
	if err != nil {
		return err
	}

	// Add document object to window
	err = windowObj.Set("document", documentObj)
	if err != nil {
		return err
	}

	// Add window object to the global scope
	err = js.vm.Set("window", windowObj)
	if err != nil {
		return err
	}

	// Make document available in the global scope directly
	err = js.vm.Set("document", documentObj)
	if err != nil {
		return err
	}

	return nil
}

// elementToJSValue converts a goquery Selection to a JavaScript object
func (js *JSEngine) elementToJSValue(selection *goquery.Selection) goja.Value {
	elementObj := js.vm.NewObject()

	// Get element ID
	if id, exists := selection.Attr("id"); exists {
		elementObj.Set("id", id)
	}

	// Set innerHTML property
	if html, err := selection.Html(); err == nil {
		elementObj.Set("innerHTML", html)

		// Add setter for innerHTML to modify the actual DOM
		elementObj.DefineAccessorProperty("innerHTML", js.vm.ToValue(func(call goja.FunctionCall) goja.Value {
			return elementObj.Get("innerHTML")
		}), js.vm.ToValue(func(call goja.FunctionCall) goja.Value {
			newHTML := call.Argument(0).String()
			selection.SetHtml(newHTML)
			return goja.Undefined()
		}), goja.FLAG_FALSE, goja.FLAG_TRUE)
	}

	// Set innerText property
	elementObj.Set("innerText", selection.Text())

	// Set getAttribute method
	elementObj.Set("getAttribute", js.vm.ToValue(func(name string) goja.Value {
		if val, exists := selection.Attr(name); exists {
			return js.vm.ToValue(val)
		}
		return goja.Null()
	}))

	// Set setAttribute method
	elementObj.Set("setAttribute", js.vm.ToValue(func(name, value string) {
		selection.SetAttr(name, value)
	}))

	return elementObj
}

// selectionToJSArray converts a goquery Selection to a JavaScript array
func (js *JSEngine) selectionToJSArray(selection *goquery.Selection) goja.Value {
	arr := js.vm.NewArray(selection.Length())

	selection.Each(func(i int, s *goquery.Selection) {
		arr.Set(fmt.Sprintf("%d", i), js.elementToJSValue(s))
	})

	return arr
}
