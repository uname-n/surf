package browser

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/headzoo/ut"
)

func TestDownload(t *testing.T) {
	ut.Run(t)

	// Create a test server that serves dummy images
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Respond with dummy image data
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write([]byte("dummy image data"))
	}))
	defer imageServer.Close()

	out := &bytes.Buffer{}
	u, _ := url.Parse(imageServer.URL)
	asset := NewImageAsset(u, "", "", "")
	l, err := DownloadAsset(asset, out)
	ut.AssertNil(err)
	ut.AssertGreaterThan(0, int(l))
	ut.AssertEquals(int(l), out.Len())
}

func TestDownloadAsync(t *testing.T) {
	ut.Run(t)

	// Create a test server that serves dummy images
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Respond with dummy image data
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write([]byte("dummy image data for " + r.URL.Path))
	}))
	defer imageServer.Close()

	ch := make(AsyncDownloadChannel, 1)
	u1, _ := url.Parse(imageServer.URL + "/image1.jpg")
	u2, _ := url.Parse(imageServer.URL + "/image2.jpg")
	asset1 := NewImageAsset(u1, "", "", "")
	asset2 := NewImageAsset(u2, "", "", "")
	out1 := &bytes.Buffer{}
	out2 := &bytes.Buffer{}

	queue := 2
	DownloadAssetAsync(asset1, out1, ch)
	DownloadAssetAsync(asset2, out2, ch)

	for {
		select {
		case result := <-ch:
			ut.AssertGreaterThan(0, int(result.Size))
			if result.Asset == asset1 {
				ut.AssertEquals(int(result.Size), out1.Len())
			} else if result.Asset == asset2 {
				ut.AssertEquals(int(result.Size), out2.Len())
			} else {
				t.Failed()
			}
			queue--
			if queue == 0 {
				goto COMPLETE
			}
		}
	}

COMPLETE:
	close(ch)
	ut.AssertEquals(0, queue)
}
