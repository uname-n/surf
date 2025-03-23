package browser

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/headzoo/ut"
	"github.com/uname-n/surf/jar"
)

func TestJavaScript(t *testing.T) {
	ut.Run(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprint(w, `<!doctype html>
<html>
	<head>
		<title>JavaScript Test</title>
	</head>
	<body>
		<div id="result">Before JS</div>
		<script type="text/javascript">
			document.getElementById("result").innerHTML = "After JS";
		</script>
	</body>
</html>`)
	}))
	defer ts.Close()

	// Create a new browser with JavaScript disabled
	bow := &Browser{}
	bow.SetUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	bow.SetState(&jar.State{})
	bow.headers = make(http.Header)
	bow.attributes = make(AttributeMap)
	bow.history = jar.NewMemoryHistory()
	bow.SetAttributes(AttributeMap{
		JavaScriptEnabled: false,
	})

	// First test with JavaScript disabled
	err := bow.Open(ts.URL)
	ut.AssertNil(err)
	ut.AssertEquals("JavaScript Test", bow.Title())
	resultText := bow.Find("#result").Text()
	ut.AssertEquals("Before JS", resultText)

	// Enable JavaScript and reload
	bow.SetJavaScriptEnabled(true)
	err = bow.Open(ts.URL)
	ut.AssertNil(err)
	resultText = bow.Find("#result").Text()
	ut.AssertEquals("After JS", resultText)
}

func TestJavaScriptExternalScript(t *testing.T) {
	ut.Run(t)

	// Create test servers
	scriptServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprint(w, `
			function updateResult() {
				document.getElementById("result").innerHTML = "External JS executed";
			}
			updateResult();
		`)
	}))
	defer scriptServer.Close()

	pageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, `<!doctype html>
<html>
	<head>
		<title>External JavaScript Test</title>
	</head>
	<body>
		<div id="result">Before JS</div>
		<script src="%s"></script>
	</body>
</html>`, scriptServer.URL)
	}))
	defer pageServer.Close()

	// Create a new browser with JavaScript enabled
	bow := &Browser{}
	bow.SetUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	bow.SetState(&jar.State{})
	bow.headers = make(http.Header)
	bow.attributes = make(AttributeMap)
	bow.history = jar.NewMemoryHistory()
	bow.SetAttributes(AttributeMap{
		JavaScriptEnabled: true,
	})

	// Test with external JavaScript
	err := bow.Open(pageServer.URL)
	ut.AssertNil(err)
	ut.AssertEquals("External JavaScript Test", bow.Title())
	resultText := bow.Find("#result").Text()
	ut.AssertEquals("External JS executed", resultText)
}
