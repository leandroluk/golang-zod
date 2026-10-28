package z

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json/v2"
	"math/big"
	"net/netip"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	// Precompiled regexes matching Zod v4 standards
	emailRegex = regexp.MustCompile(`^(?i)[a-z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$`)
	uuidRegex  = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	guidRegex  = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	ulidRegex  = regexp.MustCompile(`^[0-9A-HJ-KM-NP-TV-Z]{26}$`)
	nanoidReg  = regexp.MustCompile(`^[A-Za-z0-9_-]{21}$`)
	cuid2Reg   = regexp.MustCompile(`^[a-z][a-z0-9]{23,31}$`)
	xidReg     = regexp.MustCompile(`^[0-9a-v]{20}$`)
	ksuidReg   = regexp.MustCompile(`^[0-9A-Za-z]{27}$`)
	e164Reg    = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
	macReg     = regexp.MustCompile(`^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`)
	hostReg    = regexp.MustCompile(`^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])(\.([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9]))*$`)
	currReg    = regexp.MustCompile(`^[A-Z]{3}$`)
)

// @spec REQ-PRIM-003
// UUID accepts any valid 8-4-4-4-12 UUID.
func UUID() *StringSchema {
	return String().WithCustomCheck("uuid", func(s string) bool {
		return uuidRegex.MatchString(s)
	})
}

// @spec REQ-PRIM-003
// UUIDv4 validates UUID version 4.
func UUIDv4() *StringSchema {
	return String().WithCustomCheck("uuid_v4", func(s string) bool {
		return uuidRegex.MatchString(s) && len(s) == 36 && s[14] == '4'
	})
}

// @spec REQ-PRIM-003
// UUIDv6 validates UUID version 6.
func UUIDv6() *StringSchema {
	return String().WithCustomCheck("uuid_v6", func(s string) bool {
		return uuidRegex.MatchString(s) && len(s) == 36 && s[14] == '6'
	})
}

// @spec REQ-PRIM-003
// UUIDv7 validates UUID version 7.
func UUIDv7() *StringSchema {
	return String().WithCustomCheck("uuid_v7", func(s string) bool {
		return uuidRegex.MatchString(s) && len(s) == 36 && s[14] == '7'
	})
}

// @spec REQ-PRIM-003
// GUID validates loose GUID/UUID hex format.
func GUID() *StringSchema {
	return String().WithCustomCheck("guid", func(s string) bool {
		return guidRegex.MatchString(s)
	})
}

// @spec REQ-PRIM-003
// ULID validates Crockford Base32 26-char ULID.
func ULID() *StringSchema {
	return String().WithCustomCheck("ulid", func(s string) bool {
		return ulidRegex.MatchString(s)
	})
}

// @spec REQ-PRIM-003
// NanoID validates standard 21-char NanoID.
func NanoID() *StringSchema {
	return String().WithCustomCheck("nanoid", func(s string) bool {
		return nanoidReg.MatchString(s)
	})
}

// @spec REQ-PRIM-003
// CUID2 validates CUID version 2 format.
func CUID2() *StringSchema {
	return String().WithCustomCheck("cuid2", func(s string) bool {
		return cuid2Reg.MatchString(s)
	})
}

// @spec REQ-PRIM-003
// XID validates 20-char globally unique XID.
func XID() *StringSchema {
	return String().WithCustomCheck("xid", func(s string) bool {
		return xidReg.MatchString(s)
	})
}

// @spec REQ-PRIM-003
// KSUID validates 27-char KSUID.
func KSUID() *StringSchema {
	return String().WithCustomCheck("ksuid", func(s string) bool {
		return ksuidReg.MatchString(s)
	})
}

// @spec REQ-PRIM-004
// Email validates email addresses matching standard RFC/Zod specification.
func Email() *StringSchema {
	return String().WithCustomCheck("email", func(s string) bool {
		return emailRegex.MatchString(s)
	})
}

// @spec REQ-PRIM-004
// URL validates standard URLs with scheme and host.
func URL() *StringSchema {
	return String().WithCustomCheck("url", func(s string) bool {
		u, err := url.ParseRequestURI(s)
		return err == nil && u.Scheme != "" && u.Host != ""
	})
}

// @spec REQ-PRIM-004
// HTTPURL validates URLs with http or https scheme.
func HTTPURL() *StringSchema {
	return String().WithCustomCheck("http_url", func(s string) bool {
		u, err := url.ParseRequestURI(s)
		return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
	})
}

// @spec REQ-PRIM-004
// Hostname validates RFC 1123 hostnames.
func Hostname() *StringSchema {
	return String().WithCustomCheck("hostname", func(s string) bool {
		return len(s) <= 253 && hostReg.MatchString(s)
	})
}

// @spec REQ-PRIM-004
// IPv4 validates valid IPv4 addresses.
func IPv4() *StringSchema {
	return String().WithCustomCheck("ipv4", func(s string) bool {
		addr, err := netip.ParseAddr(s)
		return err == nil && addr.Is4()
	})
}

// @spec REQ-PRIM-004
// IPv6 validates valid IPv6 addresses.
func IPv6() *StringSchema {
	return String().WithCustomCheck("ipv6", func(s string) bool {
		addr, err := netip.ParseAddr(s)
		return err == nil && addr.Is6()
	})
}

// @spec REQ-PRIM-004
// MAC validates IEEE 802 MAC addresses.
func MAC() *StringSchema {
	return String().WithCustomCheck("mac", func(s string) bool {
		return macReg.MatchString(s)
	})
}

// @spec REQ-PRIM-004
// CIDRv4 validates IPv4 CIDR blocks.
func CIDRv4() *StringSchema {
	return String().WithCustomCheck("cidrv4", func(s string) bool {
		prefix, err := netip.ParsePrefix(s)
		return err == nil && prefix.Addr().Is4()
	})
}

// @spec REQ-PRIM-004
// CIDRv6 validates IPv6 CIDR blocks.
func CIDRv6() *StringSchema {
	return String().WithCustomCheck("cidrv6", func(s string) bool {
		prefix, err := netip.ParsePrefix(s)
		return err == nil && prefix.Addr().Is6()
	})
}

// @spec REQ-PRIM-005
// Base64 validates standard base64 encoded strings.
func Base64() *StringSchema {
	return String().WithCustomCheck("base64", func(s string) bool {
		_, err := base64.StdEncoding.DecodeString(s)
		return err == nil
	})
}

// @spec REQ-PRIM-005
// Base64URL validates unpadded URL-safe base64 strings.
func Base64URL() *StringSchema {
	return String().WithCustomCheck("base64url", func(s string) bool {
		_, err := base64.RawURLEncoding.DecodeString(s)
		return err == nil
	})
}

// @spec REQ-PRIM-005
// Hex validates hexadecimal strings with even length.
func Hex() *StringSchema {
	return String().WithCustomCheck("hex", func(s string) bool {
		if len(s)%2 != 0 {
			return false
		}
		_, err := hex.DecodeString(s)
		return err == nil
	})
}

// @spec REQ-PRIM-005
// JWT validates JSON Web Token structure (3 base64url parts, header with "alg").
func JWT() *StringSchema {
	return String().WithCustomCheck("jwt", func(s string) bool {
		parts := strings.Split(s, ".")
		if len(parts) != 3 {
			return false
		}
		headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
		if err != nil {
			return false
		}
		var header map[string]any
		if err := json.Unmarshal(headerBytes, &header); err != nil {
			return false
		}
		alg, ok := header["alg"].(string)
		return ok && alg != ""
	})
}

// @spec REQ-PRIM-005
// Emoji validates strings containing emoji runes.
func Emoji() *StringSchema {
	return String().WithCustomCheck("emoji", func(s string) bool {
		hasEmoji := false
		for _, r := range s {
			if unicode.Is(unicode.So, r) || (r >= 0x1F600 && r <= 0x1F64F) || (r >= 0x1F300 && r <= 0x1F5FF) || (r >= 0x1F680 && r <= 0x1F6FF) {
				hasEmoji = true
			}
		}
		return hasEmoji
	})
}

// @spec REQ-PRIM-005
// E164 validates E.164 phone numbers (e.g. +1234567890).
func E164() *StringSchema {
	return String().WithCustomCheck("e164", func(s string) bool {
		return e164Reg.MatchString(s)
	})
}

// @spec REQ-PRIM-005
// CreditCard validates credit card numbers using the Luhn algorithm.
func CreditCard() *StringSchema {
	return String().WithCustomCheck("credit_card", func(s string) bool {
		digits := strings.ReplaceAll(strings.ReplaceAll(s, "-", ""), " ", "")
		if len(digits) < 13 || len(digits) > 19 {
			return false
		}
		sum := 0
		alternate := false
		for i := len(digits) - 1; i >= 0; i-- {
			n, err := strconv.Atoi(string(digits[i]))
			if err != nil {
				return false
			}
			if alternate {
				n *= 2
				if n > 9 {
					n -= 9
				}
			}
			sum += n
			alternate = !alternate
		}
		return sum%10 == 0
	})
}

// @spec REQ-PRIM-005
// IBAN validates International Bank Account Numbers via mod-97 check.
func IBAN() *StringSchema {
	return String().WithCustomCheck("iban", func(s string) bool {
		cleaned := strings.ToUpper(strings.ReplaceAll(s, " ", ""))
		if len(cleaned) < 15 || len(cleaned) > 34 {
			return false
		}
		// Move first 4 characters to the end
		rearranged := cleaned[4:] + cleaned[:4]
		var numeric strings.Builder
		for _, r := range rearranged {
			if unicode.IsDigit(r) {
				numeric.WriteRune(r)
			} else if unicode.IsLetter(r) {
				numeric.WriteString(strconv.Itoa(int(r - 'A' + 10)))
			} else {
				return false
			}
		}
		n := new(big.Int)
		if _, ok := n.SetString(numeric.String(), 10); !ok {
			return false
		}
		remainder := new(big.Int).Mod(n, big.NewInt(97))
		return remainder.Int64() == 1
	})
}

// @spec REQ-PRIM-005
// CurrencyCode validates ISO 4217 3-letter currency codes.
func CurrencyCode() *StringSchema {
	return String().WithCustomCheck("currency_code", func(s string) bool {
		return currReg.MatchString(s)
	})
}

// @spec REQ-PRIM-005
// StringFormat creates a custom named string format validator.
func StringFormat(name string, check func(string) bool) *StringSchema {
	return String().WithCustomCheck(name, check)
}
