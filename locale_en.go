package z

import (
	"fmt"
	"strings"
)

type localeEN struct{}

// @spec REQ-CORE-009
var LocaleEN Locale = &localeEN{}

func (l *localeEN) Tag() string {
	return "en"
}

// @spec REQ-CORE-008
func (l *localeEN) FormatMessage(issue *Issue) string {
	switch issue.Code {
	case InvalidType:
		if issue.Expected != "" {
			return fmt.Sprintf("Expected %s", issue.Expected)
		}
		return "Invalid input"

	case TooBig:
		limit := formatValue(issue.Maximum)
		exact := issue.Exact
		inclusive := issue.Inclusive

		switch issue.Origin {
		case "string":
			if exact {
				return fmt.Sprintf("String must contain exactly %s character(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("String must contain at most %s character(s)", limit)
			}
			return fmt.Sprintf("String must contain less than %s character(s)", limit)

		case "array":
			if exact {
				return fmt.Sprintf("Array must contain exactly %s element(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("Array must contain at most %s element(s)", limit)
			}
			return fmt.Sprintf("Array must contain less than %s element(s)", limit)

		case "set":
			if exact {
				return fmt.Sprintf("Set must contain exactly %s element(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("Set must contain at most %s element(s)", limit)
			}
			return fmt.Sprintf("Set must contain less than %s element(s)", limit)

		case "map":
			if exact {
				return fmt.Sprintf("Map must contain exactly %s entry/entries", limit)
			}
			if inclusive {
				return fmt.Sprintf("Map must contain at most %s entry/entries", limit)
			}
			return fmt.Sprintf("Map must contain less than %s entry/entries", limit)

		case "bytes", "file":
			if exact {
				return fmt.Sprintf("File/bytes must contain exactly %s byte(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("File/bytes must contain at most %s byte(s)", limit)
			}
			return fmt.Sprintf("File/bytes must contain less than %s byte(s)", limit)

		case "date":
			if inclusive {
				return fmt.Sprintf("Date must be before or equal to %s", limit)
			}
			return fmt.Sprintf("Date must be before %s", limit)

		default: // number, int, bigint
			if inclusive {
				return fmt.Sprintf("Number must be less than or equal to %s", limit)
			}
			return fmt.Sprintf("Number must be less than %s", limit)
		}

	case TooSmall:
		limit := formatValue(issue.Minimum)
		exact := issue.Exact
		inclusive := issue.Inclusive

		switch issue.Origin {
		case "string":
			if exact {
				return fmt.Sprintf("String must contain exactly %s character(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("String must contain at least %s character(s)", limit)
			}
			return fmt.Sprintf("String must contain more than %s character(s)", limit)

		case "array":
			if exact {
				return fmt.Sprintf("Array must contain exactly %s element(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("Array must contain at least %s element(s)", limit)
			}
			return fmt.Sprintf("Array must contain more than %s element(s)", limit)

		case "set":
			if exact {
				return fmt.Sprintf("Set must contain exactly %s element(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("Set must contain at least %s element(s)", limit)
			}
			return fmt.Sprintf("Set must contain more than %s element(s)", limit)

		case "map":
			if exact {
				return fmt.Sprintf("Map must contain exactly %s entry/entries", limit)
			}
			if inclusive {
				return fmt.Sprintf("Map must contain at least %s entry/entries", limit)
			}
			return fmt.Sprintf("Map must contain more than %s entry/entries", limit)

		case "bytes", "file":
			if exact {
				return fmt.Sprintf("File/bytes must contain exactly %s byte(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("File/bytes must contain at least %s byte(s)", limit)
			}
			return fmt.Sprintf("File/bytes must contain more than %s byte(s)", limit)

		case "date":
			if inclusive {
				return fmt.Sprintf("Date must be after or equal to %s", limit)
			}
			return fmt.Sprintf("Date must be after %s", limit)

		default: // number, int, bigint
			if inclusive {
				return fmt.Sprintf("Number must be greater than or equal to %s", limit)
			}
			return fmt.Sprintf("Number must be greater than %s", limit)
		}

	case InvalidFormat:
		format := issue.Format
		if format == "" {
			format = "input"
		}
		if issue.Pattern != "" {
			return fmt.Sprintf("Invalid string: must match regex %s", issue.Pattern)
		}
		if issue.Prefix != "" {
			return fmt.Sprintf("Invalid string: must start with %q", issue.Prefix)
		}
		if issue.Suffix != "" {
			return fmt.Sprintf("Invalid string: must end with %q", issue.Suffix)
		}
		if issue.Includes != "" {
			return fmt.Sprintf("Invalid string: must include %q", issue.Includes)
		}
		return fmt.Sprintf("Invalid %s", format)

	case NotMultipleOf:
		return fmt.Sprintf("Number must be a multiple of %s", formatValue(issue.Divisor))

	case UnrecognizedKeys:
		return fmt.Sprintf("Unrecognized key(s) in object: %s", strings.Join(issue.Keys, ", "))

	case InvalidUnion:
		return "Invalid input matching union"

	case InvalidKey:
		return fmt.Sprintf("Invalid key in %s", issue.Origin)

	case InvalidElement:
		return fmt.Sprintf("Invalid element in %s", issue.Origin)

	case InvalidValue:
		return "Invalid value"

	case Custom:
		return "Invalid input"

	default:
		return "Validation error"
	}
}
