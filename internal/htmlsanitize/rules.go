package htmlsanitize

import "strings"

func StrictPolicy() Policy {
	el := map[string]bool{
		"p": true, "br": true, "hr": true,
		"b": true, "i": true, "u": true, "em": true, "strong": true,
		"ul": true, "ol": true, "li": true,
		"blockquote": true, "pre": true, "code": true,
	}
	return Policy{
		Elements:      el,
		Attrs:         map[string]map[string]bool{},
		Schemes:       map[string]bool{},
		StripComments: true,
	}
}

func PermissivePolicy() Policy {
	p := DefaultPolicy()
	p.RequireRelNofollow = false
	p.Elements["video"] = true
	p.Elements["audio"] = true
	p.Elements["source"] = true
	p.Elements["iframe"] = true
	p.Attrs["iframe"] = map[string]bool{
		"src": true, "width": true, "height": true,
		"frameborder": true, "allowfullscreen": true,
	}
	p.Attrs["video"] = map[string]bool{"src": true, "controls": true, "width": true, "height": true}
	p.Attrs["audio"] = map[string]bool{"src": true, "controls": true}
	return p
}

func TextOnlyPolicy() Policy {
	p := Policy{
		Elements:      map[string]bool{},
		Attrs:         map[string]map[string]bool{},
		Schemes:       map[string]bool{},
		StripComments: true,
	}
	p.Elements["html"] = false
	p.Elements["body"] = false
	p.Elements["head"] = false
	return p
}

type PolicyBuilder struct {
	p Policy
}

func NewPolicyBuilder() *PolicyBuilder {
	return &PolicyBuilder{p: Policy{
		Elements: make(map[string]bool),
		Attrs:    make(map[string]map[string]bool),
		Schemes:  make(map[string]bool),
	}}
}

func (b *PolicyBuilder) AllowElements(tags ...string) *PolicyBuilder {
	for _, t := range tags {
		b.p.Elements[strings.ToLower(t)] = true
	}
	return b
}

func (b *PolicyBuilder) AllowAttrs(element string, attrs ...string) *PolicyBuilder {
	el := strings.ToLower(element)
	if b.p.Attrs[el] == nil {
		b.p.Attrs[el] = make(map[string]bool)
	}
	for _, a := range attrs {
		b.p.Attrs[el][a] = true
	}
	return b
}

func (b *PolicyBuilder) AllowSchemes(schemes ...string) *PolicyBuilder {
	for _, s := range schemes {
		b.p.Schemes[strings.ToLower(s)] = true
	}
	return b
}

func (b *PolicyBuilder) StripComments(strip bool) *PolicyBuilder {
	b.p.StripComments = strip
	return b
}

func (b *PolicyBuilder) RequireNofollow(req bool) *PolicyBuilder {
	b.p.RequireRelNofollow = req
	return b
}

func (b *PolicyBuilder) AllowDataImages(allow bool) *PolicyBuilder {
	b.p.AllowDataImages = allow
	return b
}

func (b *PolicyBuilder) Build() Policy { return b.p }
