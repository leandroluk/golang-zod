package z

import (
	"fmt"
	"strings"
)

type localePtBR struct{}

// @spec REQ-CORE-009
var LocalePtBR Locale = &localePtBR{}

func (l *localePtBR) Tag() string {
	return "pt-BR"
}

// @spec REQ-CORE-008
func (l *localePtBR) FormatMessage(issue *Issue) string {
	switch issue.Code {
	case InvalidType:
		if issue.Expected != "" {
			return fmt.Sprintf("Esperado %s", issue.Expected)
		}
		return "Entrada inválida"

	case TooBig:
		limit := formatValue(issue.Maximum)
		exact := issue.Exact
		inclusive := issue.Inclusive

		switch issue.Origin {
		case "string":
			if exact {
				return fmt.Sprintf("String deve conter exatamente %s caractere(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("String deve conter no máximo %s caractere(s)", limit)
			}
			return fmt.Sprintf("String deve conter menos de %s caractere(s)", limit)

		case "array":
			if exact {
				return fmt.Sprintf("Array deve conter exatamente %s elemento(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("Array deve conter no máximo %s elemento(s)", limit)
			}
			return fmt.Sprintf("Array deve conter menos de %s elemento(s)", limit)

		case "set":
			if exact {
				return fmt.Sprintf("Conjunto deve conter exatamente %s elemento(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("Conjunto deve conter no máximo %s elemento(s)", limit)
			}
			return fmt.Sprintf("Conjunto deve conter menos de %s elemento(s)", limit)

		case "map":
			if exact {
				return fmt.Sprintf("Mapa deve conter exatamente %s entrada(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("Mapa deve conter no máximo %s entrada(s)", limit)
			}
			return fmt.Sprintf("Mapa deve conter menos de %s entrada(s)", limit)

		case "bytes", "file":
			if exact {
				return fmt.Sprintf("Arquivo/bytes deve conter exatamente %s byte(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("Arquivo/bytes deve conter no máximo %s byte(s)", limit)
			}
			return fmt.Sprintf("Arquivo/bytes deve conter menos de %s byte(s)", limit)

		case "date":
			if inclusive {
				return fmt.Sprintf("Data deve ser anterior ou igual a %s", limit)
			}
			return fmt.Sprintf("Data deve ser anterior a %s", limit)

		default: // number, int, bigint
			if inclusive {
				return fmt.Sprintf("Número deve ser menor ou igual a %s", limit)
			}
			return fmt.Sprintf("Número deve ser menor que %s", limit)
		}

	case TooSmall:
		limit := formatValue(issue.Minimum)
		exact := issue.Exact
		inclusive := issue.Inclusive

		switch issue.Origin {
		case "string":
			if exact {
				return fmt.Sprintf("String deve conter exatamente %s caractere(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("String deve conter no mínimo %s caractere(s)", limit)
			}
			return fmt.Sprintf("String deve conter mais de %s caractere(s)", limit)

		case "array":
			if exact {
				return fmt.Sprintf("Array deve conter exatamente %s elemento(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("Array deve conter no mínimo %s elemento(s)", limit)
			}
			return fmt.Sprintf("Array deve conter mais de %s elemento(s)", limit)

		case "set":
			if exact {
				return fmt.Sprintf("Conjunto deve conter exatamente %s elemento(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("Conjunto deve conter no mínimo %s elemento(s)", limit)
			}
			return fmt.Sprintf("Conjunto deve conter mais de %s elemento(s)", limit)

		case "map":
			if exact {
				return fmt.Sprintf("Mapa deve conter exatamente %s entrada(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("Mapa deve conter no mínimo %s entrada(s)", limit)
			}
			return fmt.Sprintf("Mapa deve conter mais de %s entrada(s)", limit)

		case "bytes", "file":
			if exact {
				return fmt.Sprintf("Arquivo/bytes deve conter exatamente %s byte(s)", limit)
			}
			if inclusive {
				return fmt.Sprintf("Arquivo/bytes deve conter no mínimo %s byte(s)", limit)
			}
			return fmt.Sprintf("Arquivo/bytes deve conter mais de %s byte(s)", limit)

		case "date":
			if inclusive {
				return fmt.Sprintf("Data deve ser posterior ou igual a %s", limit)
			}
			return fmt.Sprintf("Data deve ser posterior a %s", limit)

		default: // number, int, bigint
			if inclusive {
				return fmt.Sprintf("Número deve ser maior ou igual a %s", limit)
			}
			return fmt.Sprintf("Número deve ser maior que %s", limit)
		}

	case InvalidFormat:
		format := issue.Format
		if format == "" {
			format = "entrada"
		}
		if issue.Pattern != "" {
			return fmt.Sprintf("String inválida: deve corresponder ao padrão %s", issue.Pattern)
		}
		if issue.Prefix != "" {
			return fmt.Sprintf("String inválida: deve começar com %q", issue.Prefix)
		}
		if issue.Suffix != "" {
			return fmt.Sprintf("String inválida: deve terminar com %q", issue.Suffix)
		}
		if issue.Includes != "" {
			return fmt.Sprintf("String inválida: deve conter %q", issue.Includes)
		}
		return fmt.Sprintf("Formato inválido para %s", format)

	case NotMultipleOf:
		return fmt.Sprintf("Número deve ser um múltiplo de %s", formatValue(issue.Divisor))

	case UnrecognizedKeys:
		return fmt.Sprintf("Chave(s) não reconhecida(s) no objeto: %s", strings.Join(issue.Keys, ", "))

	case InvalidUnion:
		return "Entrada inválida para a união"

	case InvalidKey:
		return fmt.Sprintf("Chave inválida em %s", issue.Origin)

	case InvalidElement:
		return fmt.Sprintf("Elemento inválido em %s", issue.Origin)

	case InvalidValue:
		return "Valor inválido"

	case Custom:
		return "Entrada inválida"

	default:
		return "Erro de validação"
	}
}
