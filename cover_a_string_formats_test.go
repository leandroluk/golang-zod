package z_test

import (
	"context"
	"regexp"
	"slices"
	"strings"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

type coverAFormatCase struct {
	name    string
	schema  func() *z.StringSchema
	format  string
	valid   []string
	invalid []string
}

func coverARunFormats(t *testing.T, cases []coverAFormatCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.schema()
			for _, in := range tc.valid {
				if v, err := s.Parse(in); err != nil || v != in {
					t.Errorf("%q should be valid, got (%q, %v)", in, v, err)
				}
			}
			for _, in := range tc.invalid {
				_, err := s.Parse(in)
				if err == nil {
					t.Errorf("%q should be invalid", in)
					continue
				}
				is := coverAOneIssue(t, err)
				if is.Code != z.InvalidFormat || is.Format != tc.format {
					t.Errorf("%q: unexpected issue %+v", in, is)
				}
			}
		})
	}
}

// @spec REQ-PRIM-003
func TestCoverAIdentifierFormats(t *testing.T) {
	const (
		v4 = "123e4567-e89b-42d3-a456-426614174000"
		v6 = "1ef21d2f-1234-6abc-8def-123456789abc"
		v7 = "018f2d9a-1234-7abc-8def-123456789abc"
	)
	coverARunFormats(t, []coverAFormatCase{
		{"UUID", z.UUID, "uuid", []string{v4, v6, v7}, []string{"", "not-a-uuid", v4 + "0"}},
		{"UUIDv4", z.UUIDv4, "uuid_v4", []string{v4}, []string{v6, v7, "x"}},
		{"UUIDv6", z.UUIDv6, "uuid_v6", []string{v6}, []string{v4, v7, "x"}},
		{"UUIDv7", z.UUIDv7, "uuid_v7", []string{v7}, []string{v4, v6, "x"}},
		{"GUID", z.GUID, "guid", []string{v4, "00000000-0000-0000-0000-000000000000"}, []string{"0000", "zzzzzzzz-0000-0000-0000-000000000000"}},
		{"ULID", z.ULID, "ulid", []string{"01ARZ3NDEKTSV4RRFFQ69G5FAV"}, []string{"01ARZ3NDEKTSV4RRFFQ69G5FAU", "short"}},
		{"NanoID", z.NanoID, "nanoid", []string{"V1StGXR8_Z5jdHi6B-myT"}, []string{"short", "V1StGXR8_Z5jdHi6B-my!"}},
		{"CUID2", z.CUID2, "cuid2", []string{"tz4a98xxat96iws9zmbrgj3a"}, []string{"Tz4a98xxat96iws9zmbrgj3a", "1z4a98xxat96iws9zmbrgj3a", "abc"}},
		{"XID", z.XID, "xid", []string{"9m4e2mr0ui3e8a215n4g"}, []string{"9m4e2mr0ui3e8a215n4w", "short"}},
		{"KSUID", z.KSUID, "ksuid", []string{"0ujsswThIGTUYm2K8FjOOfXtY1K"}, []string{"0ujsswThIGTUYm2K8FjOOfXtY1", "0ujsswThIGTUYm2K8FjOOfXtY1!"}},
	})
}

// @spec REQ-PRIM-004
func TestCoverANetworkFormats(t *testing.T) {
	coverARunFormats(t, []coverAFormatCase{
		{"Email", z.Email, "email", []string{"a.b+c@example.com", "USER@Sub.Example.org"}, []string{"plain", "a@b", "@example.com", "a@-bad.com"}},
		{"URL", z.URL, "url", []string{"https://example.com/path?q=1", "ftp://host/file"}, []string{"example.com", "/relative", "https://", ""}},
		{"HTTPURL", z.HTTPURL, "http_url", []string{"http://example.com", "https://example.com/x"}, []string{"ftp://example.com", "https://", "nope"}},
		{"Hostname", z.Hostname, "hostname", []string{"localhost", "a.b-c.example.com"}, []string{"-bad.com", "bad_host", "", strings.Repeat("a", 254)}},
		{"IPv4", z.IPv4, "ipv4", []string{"127.0.0.1", "255.255.255.255"}, []string{"256.0.0.1", "::1", "1.2.3"}},
		{"IPv6", z.IPv6, "ipv6", []string{"::1", "2001:db8::ff00:42:8329"}, []string{"127.0.0.1", "gggg::1", ""}},
		{"MAC", z.MAC, "mac", []string{"00:1A:2b:3C:4d:5E", "00-11-22-33-44-55"}, []string{"00:11:22:33:44", "00:11:22:33:44:5G", "001122334455"}},
		{"CIDRv4", z.CIDRv4, "cidrv4", []string{"10.0.0.0/8", "192.168.1.0/24"}, []string{"10.0.0.0/33", "::1/64", "10.0.0.0"}},
		{"CIDRv6", z.CIDRv6, "cidrv6", []string{"2001:db8::/32", "::1/128"}, []string{"2001:db8::/129", "10.0.0.0/8", "::1"}},
	})
}

// @spec REQ-PRIM-005
func TestCoverAEncodingFormats(t *testing.T) {
	coverARunFormats(t, []coverAFormatCase{
		{"Base64", z.Base64, "base64", []string{"aGVsbG8=", ""}, []string{"aGVsbG8", "***"}},
		{"Base64URL", z.Base64URL, "base64url", []string{"aGVsbG8", "_-8"}, []string{"aGVsbG8=", "a+b/"}},
		{"Hex", z.Hex, "hex", []string{"deadBEEF", ""}, []string{"abc", "zz"}},
		{"JWT", z.JWT, "jwt", []string{"eyJhbGciOiJIUzI1NiJ9.e30.sig"}, []string{
			"only.two",
			"!!!.e30.sig",            // header is not base64url
			"eHl6.e30.sig",           // header is not JSON ("xyz")
			"eyJhIjoxfQ.e30.sig",     // header without alg ({"a":1})
			"eyJhbGciOiIifQ.e30.sig", // header with empty alg ({"alg":""})
		}},
		{"Emoji", z.Emoji, "emoji", []string{"\U0001F600", "hi \U0001F680", "★", "\U0001F5FF"}, []string{"abc", ""}},
		{"E164", z.E164, "e164", []string{"+14155552671", "+55"}, []string{"14155552671", "+0123", "+1", "+1234567890123456"}},
		{"CreditCard", z.CreditCard, "credit_card", []string{"4111111111111111", "4111-1111 1111-1111", "5500005555555559"}, []string{
			"4111111111111112", // fails Luhn
			"411111",           // too short
			strings.Repeat("1", 20),
			"41111111111111ab", // non digit
		}},
		{"IBAN", z.IBAN, "iban", []string{"GB82WEST12345698765432", "gb82 west 1234 5698 7654 32"}, []string{
			"GB82WEST12345698765433",  // wrong checksum
			"GB82",                    // too short
			"GB82-WEST12345698765432", // illegal character
			"GB82WEST1234569876543٢",  // non ASCII digit cannot be converted
			strings.Repeat("A", 35),
		}},
		{"CurrencyCode", z.CurrencyCode, "currency_code", []string{"USD", "BRL"}, []string{"usd", "US", "USDD"}},
	})
}

// @spec REQ-PRIM-005
func TestCoverAStringFormatCustom(t *testing.T) {
	s := z.StringFormat("even_len", func(s string) bool { return len(s)%2 == 0 })
	if v, err := s.Parse("ab"); err != nil || v != "ab" {
		t.Fatalf("got %q %v", v, err)
	}
	_, err := s.Parse("abc")
	if is := coverAOneIssue(t, err); is.Code != z.InvalidFormat || is.Format != "even_len" {
		t.Errorf("unexpected issue %+v", is)
	}
	// An empty tag falls back to the schema format name (empty here), still InvalidFormat.
	_, err = z.StringFormat("", func(string) bool { return false }).Parse("x")
	if is := coverAOneIssue(t, err); is.Code != z.InvalidFormat || is.Format != "" {
		t.Errorf("unexpected issue %+v", is)
	}
}

// @spec REQ-PRIM-006
func TestCoverAISO(t *testing.T) {
	coverARunFormats(t, []coverAFormatCase{
		{"Date", z.ISO.Date, "iso_date", []string{"2024-02-29", "1999-12-31"}, []string{"2023-02-29", "2024-13-01", "2024-1-1", "24-01-01", "2023-02-30"}},
		{"Time", z.ISO.Time, "iso_time", []string{"23:59:59", "00:00:00.123", "12:30:00Z", "12:30:00+05:30"}, []string{"24:00:00", "12:60:00", "1:2:3", "12:30"}},
		{"DateTime", func() *z.StringSchema { return z.ISO.DateTime() }, "iso_datetime", []string{"2024-01-02T03:04:05Z", "2024-01-02T03:04:05.123456789+01:00"}, []string{"2024-01-02", "2024-01-02 03:04:05", "garbage", "2024-13-02T03:04:05Z"}},
		{"Duration", z.ISO.Duration, "iso_duration", []string{"P1Y2M3DT4H5M6S", "PT1.5S", "P3D", "PT10M"}, []string{"P", "PT", "", "1Y", "P1H", "PT1Y"}},
	})
}

// @spec REQ-PRIM-002
func TestCoverAStringChecks(t *testing.T) {
	re := regexp.MustCompile(`^[a-z]+$`)
	tests := []struct {
		name  string
		build func() *z.StringSchema
		input string
		check func(t *testing.T, is z.Issue)
		ok    bool
	}{
		{"min ok", func() *z.StringSchema { return z.String().Min(2) }, "ab", nil, true},
		{"min fail", func() *z.StringSchema { return z.String().Min(3) }, "ab", func(t *testing.T, is z.Issue) {
			if is.Code != z.TooSmall || is.Origin != "string" || is.Minimum != 3 || !is.Inclusive || is.Exact {
				t.Errorf("%+v", is)
			}
		}, false},
		{"nonempty fail", func() *z.StringSchema { return z.String().NonEmpty() }, "", func(t *testing.T, is z.Issue) {
			if is.Code != z.TooSmall || is.Minimum != 1 {
				t.Errorf("%+v", is)
			}
		}, false},
		{"max ok", func() *z.StringSchema { return z.String().Max(2) }, "ab", nil, true},
		{"max fail", func() *z.StringSchema { return z.String().Max(1) }, "ab", func(t *testing.T, is z.Issue) {
			if is.Code != z.TooBig || is.Origin != "string" || is.Maximum != 1 || !is.Inclusive {
				t.Errorf("%+v", is)
			}
		}, false},
		{"length ok", func() *z.StringSchema { return z.String().Length(2) }, "ab", nil, true},
		{"length fail", func() *z.StringSchema { return z.String().Length(3) }, "ab", func(t *testing.T, is z.Issue) {
			if is.Code != z.TooSmall || !is.Exact || is.Minimum != 3 {
				t.Errorf("%+v", is)
			}
		}, false},
		{"regex ok", func() *z.StringSchema { return z.String().Regex(re) }, "abc", nil, true},
		{"regex fail", func() *z.StringSchema { return z.String().Regex(re) }, "ABC", func(t *testing.T, is z.Issue) {
			if is.Code != z.InvalidFormat || is.Format != "regex" || is.Pattern != re.String() {
				t.Errorf("%+v", is)
			}
		}, false},
		{"includes ok", func() *z.StringSchema { return z.String().Includes("b") }, "abc", nil, true},
		{"includes fail", func() *z.StringSchema { return z.String().Includes("z") }, "abc", func(t *testing.T, is z.Issue) {
			if is.Format != "includes" || is.Includes != "z" {
				t.Errorf("%+v", is)
			}
		}, false},
		{"startsWith ok", func() *z.StringSchema { return z.String().StartsWith("a") }, "abc", nil, true},
		{"startsWith fail", func() *z.StringSchema { return z.String().StartsWith("z") }, "abc", func(t *testing.T, is z.Issue) {
			if is.Format != "starts_with" || is.Prefix != "z" {
				t.Errorf("%+v", is)
			}
		}, false},
		{"endsWith ok", func() *z.StringSchema { return z.String().EndsWith("c") }, "abc", nil, true},
		{"endsWith fail", func() *z.StringSchema { return z.String().EndsWith("z") }, "abc", func(t *testing.T, is z.Issue) {
			if is.Format != "ends_with" || is.Suffix != "z" {
				t.Errorf("%+v", is)
			}
		}, false},
		{"lowercase ok", func() *z.StringSchema { return z.String().Lowercase() }, "abc_1", nil, true},
		{"lowercase fail", func() *z.StringSchema { return z.String().Lowercase() }, "aBc", func(t *testing.T, is z.Issue) {
			if is.Format != "lowercase" || is.Pattern != "^[a-z0-9_]*$" {
				t.Errorf("%+v", is)
			}
		}, false},
		{"uppercase ok", func() *z.StringSchema { return z.String().Uppercase() }, "ABC_1", nil, true},
		{"uppercase fail", func() *z.StringSchema { return z.String().Uppercase() }, "AbC", func(t *testing.T, is z.Issue) {
			if is.Format != "uppercase" || is.Pattern != "^[A-Z0-9_]*$" {
				t.Errorf("%+v", is)
			}
		}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v, err := tc.build().Parse(tc.input)
			if tc.ok {
				if err != nil || v != tc.input {
					t.Errorf("got (%q, %v)", v, err)
				}
				return
			}
			tc.check(t, coverAOneIssue(t, err))
		})
	}
	if _, err := z.String().Parse(42); err == nil {
		t.Errorf("non string must fail")
	} else if is := coverAOneIssue(t, err); is.Code != z.InvalidType || is.Expected != "string" {
		t.Errorf("%+v", is)
	}
}

// @spec REQ-PRIM-002
func TestCoverAStringTransforms(t *testing.T) {
	tests := []struct {
		name   string
		schema *z.StringSchema
		in     string
		want   string
	}{
		{"trim", z.String().Trim(), "  hi \n", "hi"},
		{"lower", z.String().ToLowerCase(), "HeLLo", "hello"},
		{"upper", z.String().ToUpperCase(), "HeLLo", "HELLO"},
		{"slugify", z.String().Slugify(), "  Hello,  World -- Go 1.2!  ", "hello-world-go-1-2"},
		{"slugify leading symbols", z.String().Slugify(), "!!!abc", "abc"},
		{"chain order", z.String().Trim().ToUpperCase().Min(2), " ab ", "AB"},
		{"transform before check", z.String().ToLowerCase().Lowercase(), "ABC", "abc"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := tc.schema.Parse(tc.in); err != nil || got != tc.want {
				t.Errorf("got (%q, %v), want %q", got, err, tc.want)
			}
		})
	}
	// Builders are immutable.
	base := z.String()
	_ = base.Trim().Min(5)
	if got, err := base.Parse(" a "); err != nil || got != " a " {
		t.Errorf("base schema must not change: (%q, %v)", got, err)
	}
}

// @spec REQ-PRIM-002 REQ-PIPE-007
func TestCoverAStringCheckOptionsAndAbort(t *testing.T) {
	// Check-level message.
	_, err := z.String().Min(5, z.Message("too short")).Parse("a")
	if m := coverAOneIssue(t, err).Message; m != "too short" {
		t.Errorf("message: %q", m)
	}

	// Multiple failures are all reported by default.
	s := z.String().Min(5).StartsWith("x").Includes("q")
	_, err = s.Parse("ab")
	if n := len(coverAIssues(t, err)); n != 3 {
		t.Errorf("expected 3 issues, got %d", n)
	}

	// Abort(true) on a failing check stops evaluating further checks.
	_, err = z.String().Min(5, z.Abort(true)).StartsWith("x").Parse("ab")
	if is := coverAOneIssue(t, err); is.Code != z.TooSmall {
		t.Errorf("abort: %+v", is)
	}

	// Abort(true) on a passing check does not stop.
	_, err = z.String().Min(1, z.Abort(true)).StartsWith("x").Parse("ab")
	if is := coverAOneIssue(t, err); is.Format != "starts_with" {
		t.Errorf("abort on pass: %+v", is)
	}

	// Parse-level abort early keeps only the first issue.
	ctx := z.WithAbortEarly(context.Background())
	_, err = s.Parse("ab", ctx)
	if is := coverAOneIssue(t, err); is.Code != z.TooSmall {
		t.Errorf("abort early: %+v", is)
	}
}

// @spec REQ-CORE-017
func TestCoverAStringCancelledMidParse(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	called := false
	s := z.String().
		WithCustomCheck("cancel_now", func(string) bool { cancel(); return true }).
		WithCustomCheck("never_reached", func(string) bool { called = true; return true })
	_, err := s.Parse("x", ctx)
	if err == nil {
		t.Errorf("expected an error after cancellation")
	}
	if called {
		t.Errorf("checks after cancellation must not run")
	}
}

// @spec REQ-PRIM-002
func TestCoverAStringCustomCheckFormatName(t *testing.T) {
	s := z.String().WithCustomCheck("first", func(string) bool { return false }).WithCustomCheck("second", func(string) bool { return false })
	_, err := s.Parse("x")
	is := coverAIssues(t, err)
	if len(is) != 2 || is[0].Format != "first" || is[1].Format != "second" {
		t.Errorf("unexpected issues %+v", is)
	}
	_, err = z.String().WithCustomCheck("c", func(string) bool { return false }, z.Message("m")).Parse("x")
	if m := coverAOneIssue(t, err).Message; m != "m" {
		t.Errorf("message: %q", m)
	}
}

// @spec REQ-PRIM-009
func TestCoverAStringModifiers(t *testing.T) {
	s := z.String().Min(2)
	if o, err := s.Optional().Parse("ab"); err != nil || !o.IsSome() {
		t.Errorf("optional: %v %v", o, err)
	}
	if n, err := s.Nullable().Parse(nil); err != nil || !n.IsNull() {
		t.Errorf("nullable: %v %v", n, err)
	}
	_ = s.Nullish()
	if v, err := s.Default("dd").Parse(nil); err != nil || v != "dd" {
		t.Errorf("default: %v %v", v, err)
	}
	if v, err := s.Array().Parse([]any{"ab", "cd"}); err != nil || !slices.Equal(v, []string{"ab", "cd"}) {
		t.Errorf("array: %v %v", v, err)
	}
	m := s.Meta(func(b *z.MetaBuilder) { b.Title("name").ID("coverA_string_meta") })
	if mm := m.GetMeta(); mm.Title != "name" || mm.ID != "coverA_string_meta" {
		t.Errorf("meta: %+v", mm)
	}
	if _, err := m.Parse("a"); err == nil {
		t.Errorf("meta clone must keep checks")
	}
	if got := s.GetMeta().Title; got != "" {
		t.Errorf("Meta must not mutate the original schema, title=%q", got)
	}
}

// @spec REQ-PIPE-007
func TestCoverAStringErrorOverrides(t *testing.T) {
	s := z.String().Min(3)
	_, err := s.Error("schema msg").Parse("a")
	if m := coverAOneIssue(t, err).Message; m != "schema msg" {
		t.Errorf("Error: %q", m)
	}
	_, err = s.ErrorFunc(func(is *z.Issue) string { return "fn for " + string(is.Code) }).Parse("a")
	if m := coverAOneIssue(t, err).Message; m != "fn for too_small" {
		t.Errorf("ErrorFunc: %q", m)
	}
}

// @spec REQ-PRIM-008
func TestCoverACoerceStringKeepsCoercionAfterBuilders(t *testing.T) {
	s := z.Coerce.String().Trim().Min(2).Meta(func(b *z.MetaBuilder) { b.Title("coerced") })
	if v, err := s.Parse(123); err != nil || v != "123" {
		t.Errorf("got (%q, %v)", v, err)
	}
	_, err := z.Coerce.String().Min(3).Parse(1)
	if is := coverAOneIssue(t, err); is.Code != z.TooSmall {
		t.Errorf("unexpected issue %+v", is)
	}
}
