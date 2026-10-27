package z

import (
	"fmt"
	"strings"
)

type localeES struct{}

// @spec REQ-CORE-009
var LocaleES Locale = &localeES{}

func (l *localeES) Tag() string {
	return "es"
}

// @spec REQ-CORE-008
func (l *localeES) FormatMessage(issue *Issue) string {
	switch issue.Code {
	case InvalidType:
		if issue.Expected != "" {
			return fmt.Sprintf("Se esperaba %s", issue.Expected)
		}
		return "Entrada no válida"

	case TooBig:
		limit := formatValue(issue.Maximum)
		exact := issue.Exact
		inclusive := issue.Inclusive

		switch issue.Origin {
		case "string":
			if exact {
				return fmt.Sprintf("El texto debe contener exactamente %s carácter(es)", limit)
			}
			if inclusive {
				return fmt.Sprintf("El texto debe contener como máximo %s carácter(es)", limit)
			}
			return fmt.Sprintf("El texto debe contener menos de %s carácter(es)", limit)

		case "array":
			if exact {
				return fmt.Sprintf("El array debe contener exactamente %s elemento(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("El array debe contener como máximo %s elemento(s)", limit)
			}
			return fmt.Sprintf("El array debe contener menos de %s elemento(s)", limit)

		case "set":
			if exact {
				return fmt.Sprintf("El conjunto debe contener exactamente %s elemento(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("El conjunto debe contener como máximo %s elemento(s)", limit)
			}
			return fmt.Sprintf("El conjunto debe contener menos de %s elemento(s)", limit)

		case "map":
			if exact {
				return fmt.Sprintf("El mapa debe contener exactamente %s entrada(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("El mapa debe contener como máximo %s entrada(s)", limit)
			}
			return fmt.Sprintf("El mapa debe contener menos de %s entrada(s)", limit)

		case "bytes", "file":
			if exact {
				return fmt.Sprintf("El archivo/bytes debe contener exactamente %s byte(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("El archivo/bytes debe contener como máximo %s byte(s)", limit)
			}
			return fmt.Sprintf("El archivo/bytes debe contener menos de %s byte(s)", limit)

		case "date":
			if inclusive {
				return fmt.Sprintf("La fecha debe ser anterior o igual a %s", limit)
			}
			return fmt.Sprintf("La fecha debe ser anterior a %s", limit)

		default: // number, int, bigint
			if inclusive {
				return fmt.Sprintf("El número debe ser menor o igual a %s", limit)
			}
			return fmt.Sprintf("El número debe ser menor que %s", limit)
		}

	case TooSmall:
		limit := formatValue(issue.Minimum)
		exact := issue.Exact
		inclusive := issue.Inclusive

		switch issue.Origin {
		case "string":
			if exact {
				return fmt.Sprintf("El texto debe contener exactamente %s carácter(es)", limit)
			}
			if inclusive {
				return fmt.Sprintf("El texto debe contener al menos %s carácter(es)", limit)
			}
			return fmt.Sprintf("El texto debe contener más de %s carácter(es)", limit)

		case "array":
			if exact {
				return fmt.Sprintf("El array debe contener exactamente %s elemento(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("El array debe contener al menos %s elemento(s)", limit)
			}
			return fmt.Sprintf("El array debe contener más de %s elemento(s)", limit)

		case "set":
			if exact {
				return fmt.Sprintf("El conjunto debe contener exactamente %s elemento(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("El conjunto debe contener al menos %s elemento(s)", limit)
			}
			return fmt.Sprintf("El conjunto debe contener más de %s elemento(s)", limit)

		case "map":
			if exact {
				return fmt.Sprintf("El mapa debe contener exactamente %s entrada(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("El mapa debe contener al menos %s entrada(s)", limit)
			}
			return fmt.Sprintf("El mapa debe contener más de %s entrada(s)", limit)

		case "bytes", "file":
			if exact {
				return fmt.Sprintf("El archivo/bytes debe contener exactamente %s byte(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("El archivo/bytes debe contener al menos %s byte(s)", limit)
			}
			return fmt.Sprintf("El archivo/bytes debe contener más de %s byte(s)", limit)

		case "date":
			if inclusive {
				return fmt.Sprintf("La fecha debe ser posterior o igual a %s", limit)
			}
			return fmt.Sprintf("La fecha debe ser posterior a %s", limit)

		default: // number, int, bigint
			if inclusive {
				return fmt.Sprintf("El número debe ser mayor o igual a %s", limit)
			}
			return fmt.Sprintf("El número debe ser mayor que %s", limit)
		}

	case InvalidFormat:
		format := issue.Format
		if format == "" {
			format = "entrada"
		}
		if issue.Pattern != "" {
			return fmt.Sprintf("Texto no válido: debe coincidir con el patrón %s", issue.Pattern)
		}
		if issue.Prefix != "" {
			return fmt.Sprintf("Texto no válido: debe comenzar con %q", issue.Prefix)
		}
		if issue.Suffix != "" {
			return fmt.Sprintf("Texto no válido: debe terminar con %q", issue.Suffix)
		}
		if issue.Includes != "" {
			return fmt.Sprintf("Texto no válido: debe contener %q", issue.Includes)
		}
		return fmt.Sprintf("Formato no válido para %s", format)

	case NotMultipleOf:
		return fmt.Sprintf("El número debe ser un múltiplo de %s", formatValue(issue.Divisor))

	case UnrecognizedKeys:
		return fmt.Sprintf("Clave(s) no reconocida(s) en el objeto: %s", strings.Join(issue.Keys, ", "))

	case InvalidUnion:
		return "Entrada no válida para la unión"

	case InvalidKey:
		return fmt.Sprintf("Clave no válida en %s", issue.Origin)

	case InvalidElement:
		return fmt.Sprintf("Elemento no válido en %s", issue.Origin)

	case InvalidValue:
		return "Valor no válido"

	case Custom:
		return "Entrada no válida"

	default:
		return "Error de validación"
	}
}
