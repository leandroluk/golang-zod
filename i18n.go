package z

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// @spec REQ-CORE-009
// Locale formats error messages for issues according to language and region.
type Locale interface {
	Tag() string
	FormatMessage(issue *Issue) string
}

var (
	globalLocale atomic.Pointer[Locale]
	localeMu     sync.RWMutex
	locales      = make(map[string]Locale)
)

func init() {
	// Initialize default global locale
	var en Locale = LocaleEN
	globalLocale.Store(&en)

	// Register built-in locales
	RegisterLocale("en", LocaleEN)
	RegisterLocale("en-US", LocaleEN)
	RegisterLocale("pt-BR", LocalePtBR)
	RegisterLocale("pt", LocalePtBR)
	RegisterLocale("es", LocaleES)
	RegisterLocale("es-ES", LocaleES)
}

// @spec REQ-CORE-009
// SetLocale sets the global default locale atomically, safe for concurrent use.
func SetLocale(l Locale) {
	if l == nil {
		l = LocaleEN
	}
	globalLocale.Store(&l)
}

// @spec REQ-CORE-009
// GetLocale returns the active global default locale.
func GetLocale() Locale {
	return *globalLocale.Load()
}

// @spec REQ-CORE-014
// RegisterLocale registers a custom locale under the given language tag.
// Lookups are case-insensitive.
func RegisterLocale(tag string, l Locale) {
	localeMu.Lock()
	defer localeMu.Unlock()
	locales[strings.ToLower(strings.TrimSpace(tag))] = l
}

// @spec REQ-CORE-013
// WithLocale derives a context containing the specified locale tag or Accept-Language header.
// If ctx is nil, context.Background() is used. If tag is empty or no match is found,
// ctx is returned unchanged.
func WithLocale(ctx context.Context, tag string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return ctx
	}

	loc := ResolveLocale(tag)
	if loc == nil {
		return ctx
	}

	key := getActiveLocaleKey()
	// nolint:staticcheck // String context key is by design (ADR-017)
	return context.WithValue(ctx, key, loc)
}

// @spec REQ-CORE-013
// ResolveLocale resolves a language tag or Accept-Language header against registered locales.
func ResolveLocale(headerOrTag string) Locale {
	localeMu.RLock()
	defer localeMu.RUnlock()

	type langPref struct {
		tag string
		q   float64
	}

	var prefs []langPref
	parts := strings.Split(headerOrTag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		subparts := strings.Split(part, ";")
		t := strings.TrimSpace(subparts[0])
		q := 1.0
		for _, param := range subparts[1:] {
			param = strings.TrimSpace(param)
			if strings.HasPrefix(param, "q=") {
				if parsedQ, err := strconv.ParseFloat(param[2:], 64); err == nil {
					q = parsedQ
				}
			}
		}
		prefs = append(prefs, langPref{tag: strings.ToLower(t), q: q})
	}

	// Sort by q descending
	for i := 0; i < len(prefs); i++ {
		for j := i + 1; j < len(prefs); j++ {
			if prefs[j].q > prefs[i].q {
				prefs[i], prefs[j] = prefs[j], prefs[i]
			}
		}
	}

	// Check each preference
	for _, p := range prefs {
		// 1. Exact match
		if loc, ok := locales[p.tag]; ok {
			return loc
		}
		// 2. Prefix match (e.g. es-mx -> es, or pt -> pt-br)
		prefix := p.tag
		if dash := strings.IndexByte(prefix, '-'); dash != -1 {
			prefix = prefix[:dash]
			if loc, ok := locales[prefix]; ok {
				return loc
			}
		}
		// Check if any registered tag has this prefix
		for regTag, loc := range locales {
			if strings.HasPrefix(regTag, prefix) {
				return loc
			}
		}
	}

	return nil
}

// resolveActiveLocale determines the locale to use for a parse call.
// Hierarchy: (1) ctx.Value(activeLocaleKey) -> (2) SetLocale -> (3) LocaleEN
func resolveActiveLocale(ctx context.Context) Locale {
	if ctx != nil {
		key := getActiveLocaleKey()
		val := ctx.Value(key)
		if val != nil {
			switch v := val.(type) {
			case Locale:
				return v
			case string:
				if loc := ResolveLocale(v); loc != nil {
					return loc
				}
			}
		}
	}

	return GetLocale()
}

// resolveIssueMessage resolves the message for an issue using the active locale,
// preserving custom messages if already present.
func resolveIssueMessage(issue *Issue, loc Locale) {
	if issue.Message != "" {
		return
	}
	issue.Message = loc.FormatMessage(issue)
}

func formatValue(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}
