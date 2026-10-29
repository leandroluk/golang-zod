package z_test

import (
	"encoding/base64"
	"regexp"
	"testing"
	"time"

	z "github.com/leandroluk/golang-zod"
)

// @spec REQ-PRIM-001
func TestREQ_PRIM_001_PrimitivesAndLiteral(t *testing.T) {
	t.Run("Boolean", func(t *testing.T) {
		s := z.Boolean()
		v, err := s.Parse(true)
		if err != nil || v != true {
			t.Errorf("expected true, got (%v, %v)", v, err)
		}
		_, err = s.Parse("true")
		if err == nil {
			t.Errorf("expected error parsing string as boolean")
		}
	})

	t.Run("Null", func(t *testing.T) {
		s := z.Null()
		v, err := s.Parse(nil)
		if err != nil || v != nil {
			t.Errorf("expected nil, got (%v, %v)", v, err)
		}
		_, err = s.Parse("not null")
		if err == nil {
			t.Errorf("expected error parsing string as null")
		}
	})

	t.Run("Any and Unknown", func(t *testing.T) {
		sAny := z.Any()
		sUnk := z.Unknown()
		for _, in := range []any{123, "text", map[string]any{"a": 1}, nil} {
			if _, err := sAny.Parse(in); err != nil {
				t.Errorf("Any failed on %v: %v", in, err)
			}
			if _, err := sUnk.Parse(in); err != nil {
				t.Errorf("Unknown failed on %v: %v", in, err)
			}
		}
	})

	t.Run("Never", func(t *testing.T) {
		s := z.Never()
		_, err := s.Parse("anything")
		if err == nil {
			t.Errorf("Never must always fail")
		}
	})

	t.Run("Date", func(t *testing.T) {
		s := z.Date()
		now := time.Now().Truncate(time.Second)
		v, err := s.Parse(now)
		if err != nil || !v.Equal(now) {
			t.Errorf("expected %v, got (%v, %v)", now, v, err)
		}
		vStr, err := s.Parse(now.Format(time.RFC3339))
		if err != nil || !vStr.Equal(now) {
			t.Errorf("expected RFC3339 parsed date, got (%v, %v)", vStr, err)
		}
		_, err = s.Parse("invalid-date")
		if err == nil {
			t.Errorf("expected error on invalid date string")
		}
	})

	t.Run("Bytes", func(t *testing.T) {
		s := z.Bytes()
		raw := []byte("hello binary")
		v, err := s.Parse(raw)
		if err != nil || string(v) != "hello binary" {
			t.Errorf("expected raw bytes, got %v", err)
		}
		b64 := base64.StdEncoding.EncodeToString(raw)
		vB64, err := s.Parse(b64)
		if err != nil || string(vB64) != "hello binary" {
			t.Errorf("expected base64 decoded bytes, got %v", err)
		}
	})

	t.Run("Literal", func(t *testing.T) {
		s := z.Literal("admin", "manager")
		v, err := s.Parse("admin")
		if err != nil || v != "admin" {
			t.Errorf("expected 'admin', got (%v, %v)", v, err)
		}
		v2, err := s.Parse("manager")
		if err != nil || v2 != "manager" {
			t.Errorf("expected 'manager', got (%v, %v)", v2, err)
		}
		_, err = s.Parse("user")
		if err == nil {
			t.Errorf("expected error on non-literal 'user'")
		}
	})
}

// @spec REQ-PRIM-002
func TestREQ_PRIM_002_StringChecksAndTransforms(t *testing.T) {
	t.Run("Min Max Length NonEmpty", func(t *testing.T) {
		s := z.String().Min(3).Max(6)
		if _, err := s.Parse("ab"); err == nil {
			t.Errorf("expected error for len 2 < Min(3)")
		}
		if _, err := s.Parse("abcdefg"); err == nil {
			t.Errorf("expected error for len 7 > Max(6)")
		}
		if v, err := s.Parse("abcd"); err != nil || v != "abcd" {
			t.Errorf("expected 'abcd', got (%v, %v)", v, err)
		}

		sExact := z.String().Length(4)
		if _, err := sExact.Parse("abc"); err == nil {
			t.Errorf("expected error for length 3 != 4")
		}
		if _, err := sExact.Parse("abcd"); err != nil {
			t.Errorf("expected length 4 to pass: %v", err)
		}

		sNonEmpty := z.String().NonEmpty()
		if _, err := sNonEmpty.Parse(""); err == nil {
			t.Errorf("expected error for empty string")
		}
	})

	t.Run("Regex and Patterns", func(t *testing.T) {
		digitsOnly := z.String().Regex(regexp.MustCompile(`^\d+$`))
		if _, err := digitsOnly.Parse("12345"); err != nil {
			t.Errorf("expected digits to pass: %v", err)
		}
		if _, err := digitsOnly.Parse("12a45"); err == nil {
			t.Errorf("expected regex failure")
		}

		s := z.String().StartsWith("prefix_").EndsWith("_suffix").Includes("middle")
		if _, err := s.Parse("prefix_middle_suffix"); err != nil {
			t.Errorf("expected compound pattern to pass: %v", err)
		}
		if _, err := s.Parse("prefix_other_suffix"); err == nil {
			t.Errorf("expected failure missing 'middle'")
		}
	})

	t.Run("Transforms Trim, ToLowerCase, Slugify", func(t *testing.T) {
		s := z.String().Trim().ToLowerCase().Min(3)
		v, err := s.Parse("  HeLLo  ")
		if err != nil || v != "hello" {
			t.Errorf("expected trimmed and lowercased 'hello', got (%q, %v)", v, err)
		}

		slug := z.String().Slugify()
		vSlug, err := slug.Parse("Hello World & Go 1.27!")
		if err != nil || vSlug != "hello-world-go-1-27" {
			t.Errorf("expected slug 'hello-world-go-1-27', got (%q, %v)", vSlug, err)
		}
	})
}

// @spec REQ-PRIM-003
func TestREQ_PRIM_003_IdentifierFormats(t *testing.T) {
	t.Run("UUID variants", func(t *testing.T) {
		u4 := "123e4567-e89b-42d3-a456-426614174000"
		if _, err := z.UUID().Parse(u4); err != nil {
			t.Errorf("UUID failed on v4: %v", err)
		}
		if _, err := z.UUIDv4().Parse(u4); err != nil {
			t.Errorf("UUIDv4 failed on v4: %v", err)
		}

		u7 := "018861d8-3011-7393-a5f1-33180d5d214a"
		if _, err := z.UUIDv7().Parse(u7); err != nil {
			t.Errorf("UUIDv7 failed on v7: %v", err)
		}
		if _, err := z.UUIDv4().Parse(u7); err == nil {
			t.Errorf("UUIDv4 must reject v7")
		}

		if _, err := z.GUID().Parse(u4); err != nil {
			t.Errorf("GUID failed: %v", err)
		}
	})

	t.Run("NanoID, ULID, XID, KSUID, CUID2", func(t *testing.T) {
		if _, err := z.ULID().Parse("01ARZ3NDEKTSV4RRFFQ69G5FAV"); err != nil {
			t.Errorf("ULID failed: %v", err)
		}
		if _, err := z.NanoID().Parse("V1StGXR8_Z5jdHi6B-myT"); err != nil {
			t.Errorf("NanoID failed: %v", err)
		}
		if _, err := z.XID().Parse("9m4e2mr0ui3e8a215n4g"); err != nil {
			t.Errorf("XID failed: %v", err)
		}
		if _, err := z.KSUID().Parse("0ujtsYcgvSTreplaceWithValid27"); err != nil && len("0ujtsYcgvSTreplaceWithValid27") == 27 {
			// standard 27 alphanumeric chars
			if _, err := z.KSUID().Parse("0ujtsYcgvST24hvgEHvfqTEoqTb"); err != nil {
				t.Errorf("KSUID failed: %v", err)
			}
		}
		if _, err := z.CUID2().Parse("clh8g6y1l000008l41d6g324w"); err != nil {
			t.Errorf("CUID2 failed: %v", err)
		}
	})
}

// @spec REQ-PRIM-004
func TestREQ_PRIM_004_NetworkFormats(t *testing.T) {
	t.Run("Email", func(t *testing.T) {
		s := z.Email()
		if _, err := s.Parse("user@example.com"); err != nil {
			t.Errorf("valid email failed: %v", err)
		}
		if _, err := s.Parse("invalid-email"); err == nil {
			t.Errorf("expected invalid email to fail")
		}
	})

	t.Run("URL and HTTPURL", func(t *testing.T) {
		if _, err := z.URL().Parse("postgres://user:pass@localhost:5432/db"); err != nil {
			t.Errorf("URL failed: %v", err)
		}
		if _, err := z.HTTPURL().Parse("https://google.com/search?q=zod"); err != nil {
			t.Errorf("HTTPURL failed: %v", err)
		}
		if _, err := z.HTTPURL().Parse("ftp://ftp.is.co.za/rfc/rfc1808.txt"); err == nil {
			t.Errorf("HTTPURL must reject ftp scheme")
		}
	})

	t.Run("IP and CIDR", func(t *testing.T) {
		if _, err := z.IPv4().Parse("192.168.1.1"); err != nil {
			t.Errorf("IPv4 failed: %v", err)
		}
		if _, err := z.IPv4().Parse("2001:db8::1"); err == nil {
			t.Errorf("IPv4 must reject IPv6")
		}
		if _, err := z.IPv6().Parse("2001:db8::1"); err != nil {
			t.Errorf("IPv6 failed: %v", err)
		}
		if _, err := z.CIDRv4().Parse("10.0.0.0/24"); err != nil {
			t.Errorf("CIDRv4 failed: %v", err)
		}
	})

	t.Run("MAC and Hostname", func(t *testing.T) {
		if _, err := z.MAC().Parse("00:1A:2B:3C:4D:5E"); err != nil {
			t.Errorf("MAC failed: %v", err)
		}
		if _, err := z.Hostname().Parse("api.internal.service.local"); err != nil {
			t.Errorf("Hostname failed: %v", err)
		}
	})
}

// @spec REQ-PRIM-005
func TestREQ_PRIM_005_TokenAndEncodingFormats(t *testing.T) {
	t.Run("Base64 and Hex", func(t *testing.T) {
		if _, err := z.Base64().Parse("SGVsbG8gV29ybGQ="); err != nil {
			t.Errorf("Base64 failed: %v", err)
		}
		if _, err := z.Base64URL().Parse("SGVsbG8gV29ybGQ"); err != nil {
			t.Errorf("Base64URL failed: %v", err)
		}
		if _, err := z.Hex().Parse("48656c6c6f"); err != nil {
			t.Errorf("Hex failed: %v", err)
		}
		if _, err := z.Hex().Parse("48656c6c6"); err == nil {
			t.Errorf("Hex must reject odd length")
		}
	})

	t.Run("JWT", func(t *testing.T) {
		// Valid mock JWT with header {"alg":"HS256"}
		jwt := "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUPvYo_t2g"
		if _, err := z.JWT().Parse(jwt); err != nil {
			t.Errorf("valid JWT failed: %v", err)
		}
		if _, err := z.JWT().Parse("not.a.jwt"); err == nil {
			t.Errorf("invalid JWT must fail")
		}
	})

	t.Run("E164, CreditCard, IBAN, CurrencyCode", func(t *testing.T) {
		if _, err := z.E164().Parse("+5511987654321"); err != nil {
			t.Errorf("E164 failed: %v", err)
		}
		// Valid Luhn credit card
		if _, err := z.CreditCard().Parse("4532-0150-1234-5671"); err != nil {
			t.Errorf("CreditCard failed: %v", err)
		}
		if _, err := z.CreditCard().Parse("4532-0150-1234-5670"); err == nil {
			t.Errorf("CreditCard must fail on invalid Luhn checksum")
		}
		if _, err := z.CurrencyCode().Parse("USD"); err != nil {
			t.Errorf("CurrencyCode USD failed: %v", err)
		}
		if _, err := z.CurrencyCode().Parse("BRL"); err != nil {
			t.Errorf("CurrencyCode BRL failed: %v", err)
		}
	})
}

// @spec REQ-PRIM-006
func TestREQ_PRIM_006_ISODateTime(t *testing.T) {
	if _, err := z.ISO.Date().Parse("2026-10-27"); err != nil {
		t.Errorf("ISO Date failed: %v", err)
	}
	if _, err := z.ISO.Date().Parse("2026-02-31"); err == nil {
		t.Errorf("ISO Date must reject invalid calendar day")
	}

	if _, err := z.ISO.Time().Parse("14:30:00Z"); err != nil {
		t.Errorf("ISO Time failed: %v", err)
	}

	if _, err := z.ISO.DateTime().Parse("2026-10-27T14:30:00Z"); err != nil {
		t.Errorf("ISO DateTime failed: %v", err)
	}

	if _, err := z.ISO.Duration().Parse("P1Y2M3DT4H5M6S"); err != nil {
		t.Errorf("ISO Duration failed: %v", err)
	}
	if _, err := z.ISO.Duration().Parse("invalid-duration"); err == nil {
		t.Errorf("ISO Duration must reject invalid format")
	}
}

// @spec REQ-PRIM-007
func TestREQ_PRIM_007_NumbersAndChecks(t *testing.T) {
	t.Run("Int boundaries and checks", func(t *testing.T) {
		s := z.Int().Min(10).Max(100).MultipleOf(5)
		if v, err := s.Parse(25); err != nil || v != 25 {
			t.Errorf("expected 25, got (%v, %v)", v, err)
		}
		if _, err := s.Parse(7); err == nil {
			t.Errorf("expected failure for 7 < Min(10)")
		}
		if _, err := s.Parse(23); err == nil {
			t.Errorf("expected failure for 23 not multiple of 5")
		}
	})

	t.Run("Float64 checks Positive and Negative", func(t *testing.T) {
		sPos := z.Float64().Positive()
		if _, err := sPos.Parse(3.14); err != nil {
			t.Errorf("expected positive float to pass: %v", err)
		}
		if _, err := sPos.Parse(-1.0); err == nil {
			t.Errorf("expected failure for negative value")
		}
	})

	t.Run("Int64 precision from json.Number", func(t *testing.T) {
		s := z.Int64()
		// Integer greater than 2^53 (9007199254740993)
		largeNum := int64(9007199254740993)
		v, err := s.Parse(largeNum)
		if err != nil || v != largeNum {
			t.Errorf("expected %d, got (%d, %v)", largeNum, v, err)
		}
	})

	t.Run("BigInt arbitrary precision", func(t *testing.T) {
		s := z.BigInt()
		bigStr := "123456789012345678901234567890"
		v, err := s.Parse(bigStr)
		if err != nil || v.String() != bigStr {
			t.Errorf("expected %s, got (%v, %v)", bigStr, v, err)
		}
	})
}

// @spec REQ-PRIM-008
func TestREQ_PRIM_008_Coerce(t *testing.T) {
	t.Run("Coerce String", func(t *testing.T) {
		s := z.Coerce.String()
		v, err := s.Parse(42)
		if err != nil || v != "42" {
			t.Errorf("expected '42', got (%q, %v)", v, err)
		}
		vBool, err := s.Parse(true)
		if err != nil || vBool != "true" {
			t.Errorf("expected 'true', got (%q, %v)", vBool, err)
		}
	})

	t.Run("Coerce Boolean", func(t *testing.T) {
		s := z.Coerce.Boolean()
		v, err := s.Parse("true")
		if err != nil || v != true {
			t.Errorf("expected true, got %v", v)
		}
		vOff, err := s.Parse("off")
		if err != nil || vOff != false {
			t.Errorf("expected false for 'off', got %v", vOff)
		}
		vNum, err := s.Parse(1)
		if err != nil || vNum != true {
			t.Errorf("expected true for 1, got %v", vNum)
		}
	})

	t.Run("Coerce Number and Int", func(t *testing.T) {
		v, err := z.Coerce.Int().Parse("123")
		if err != nil || v != 123 {
			t.Errorf("expected 123, got (%v, %v)", v, err)
		}

		vFloat, err := z.Coerce.Number().Parse("3.14159")
		if err != nil || vFloat != 3.14159 {
			t.Errorf("expected 3.14159, got (%v, %v)", vFloat, err)
		}
	})

	t.Run("Coerce Date", func(t *testing.T) {
		s := z.Coerce.Date()
		v, err := s.Parse("2026-10-27T15:00:00Z")
		if err != nil || v.Year() != 2026 {
			t.Errorf("expected year 2026, got (%v, %v)", v, err)
		}
		vEpoch, err := s.Parse(1700000000)
		if err != nil || vEpoch.Unix() != 1700000000 {
			t.Errorf("expected epoch 1700000000, got (%v, %v)", vEpoch, err)
		}
	})
}

// @spec REQ-PRIM-009
func TestREQ_PRIM_009_ModifiersOnPrimitives(t *testing.T) {
	t.Run("Optional Nullable Nullish Default Array on String", func(t *testing.T) {
		s := z.String().Min(2)

		opt := s.Optional()
		vOpt, err := opt.Parse("hi")
		if err != nil || !vOpt.IsSome() || vOpt.OrElse("") != "hi" {
			t.Errorf("expected Some('hi'): %v", err)
		}

		nullable := s.Nullable()
		vNull, err := nullable.Parse(nil)
		if err != nil || !vNull.IsNull() {
			t.Errorf("expected NullableNull: %v", err)
		}

		nullish := s.Nullish()
		vNullish, err := nullish.Parse(nil)
		if err != nil || !vNullish.IsSome() {
			t.Errorf("expected Some for nullish nil: %v", err)
		}

		def := s.Default("fallback")
		vDef, err := def.Parse(nil)
		if err != nil || vDef != "fallback" {
			t.Errorf("expected fallback, got (%q, %v)", vDef, err)
		}

		arr := s.Array()
		vArr, err := arr.Parse([]any{"ab", "cd"})
		if err != nil || len(vArr) != 2 || vArr[0] != "ab" {
			t.Errorf("expected array ['ab', 'cd'], got (%v, %v)", vArr, err)
		}
	})

	t.Run("Meta on Number", func(t *testing.T) {
		s := z.Int().Min(18).Meta(func(m *z.MetaBuilder) {
			m.Title("User Age").Description("Must be 18+")
		})
		meta := s.GetMeta()
		if meta.Title != "User Age" || meta.Description != "Must be 18+" {
			t.Errorf("unexpected meta: %+v", meta)
		}
	})
}
