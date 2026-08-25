package htmlparse

import (
	"strings"

	"golang.org/x/net/html"
)

type Image struct {
	Src     string
	Alt     string
	Width   string
	Height  string
	Loading string
	Classes []string
}

func ExtractImages(d *Doc) []Image {
	if d == nil || d.Root == nil {
		return nil
	}
	var images []Image
	_ = Walk(d.Root, func(n *html.Node) error {
		if n.Type != html.ElementNode || n.Data != "img" {
			return nil
		}
		img := Image{
			Src:     Attr(n, "src"),
			Alt:     Attr(n, "alt"),
			Width:   Attr(n, "width"),
			Height:  Attr(n, "height"),
			Loading: Attr(n, "loading"),
		}
		if cls := Attr(n, "class"); cls != "" {
			img.Classes = strings.Fields(cls)
		}
		images = append(images, img)
		return nil
	})
	return images
}

func MissingAlt(images []Image) []Image {
	var out []Image
	for _, img := range images {
		if strings.TrimSpace(img.Alt) == "" {
			out = append(out, img)
		}
	}
	return out
}

func ImagesBySrc(images []Image) map[string]Image {
	m := make(map[string]Image)
	for _, img := range images {
		if img.Src != "" {
			m[img.Src] = img
		}
	}
	return m
}

func LazyLoadedCount(images []Image) int {
	n := 0
	for _, img := range images {
		if strings.ToLower(img.Loading) == "lazy" {
			n++
		}
	}
	return n
}

func ExternalImages(images []Image) []Image {
	var out []Image
	for _, img := range images {
		lower := strings.ToLower(img.Src)
		if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
			out = append(out, img)
		}
	}
	return out
}

func DataImages(images []Image) []Image {
	var out []Image
	for _, img := range images {
		if strings.HasPrefix(strings.ToLower(img.Src), "data:") {
			out = append(out, img)
		}
	}
	return out
}
