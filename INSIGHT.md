# INSIGHT.md — Paradigmas do Go, Arquitetura sem Reflexão em Runtime e API Pura do Zod v4 (`golang-zod`)

Baseado na inspeção direta do código-fonte do **Zod v4** (`colinhacks/zod`, `packages/zod/src/v4/`) e adaptado estritamente aos padrões e limites idiomáticos da linguagem **Go (Golang)**, este documento estabelece a fundação técnica do **`golang-zod`**:
* **Apenas API Nativa do Zod v4** (Zero legado do Zod v3).
* **Paradigmas Duais**: **Schema-First** (`z.Object(z.Shape{...})`) e **Type-First** (`z.Object[User]()`).
* **Compilação da AST com Cache**: Reflexão (`reflect`) permitida estritamente no bootstrap/startup para montar a árvore imutável; **Zero reflexão no caminho crítico de validação**.
* **Erros como Valores & i18n Nativo**: Formatação estruturada (`Format`, `Flatten`, `Treeify`, `Prettify`) e suporte multi-idioma (`en`, `pt-BR`, etc.).
* **Concorrência Nativa & `sync.Pool`**: Schemas imutáveis seguros para concorrência de goroutines com reaproveitamento de buffers via pool.

---

## 1. Limpeza Radical: Apenas API Nativa do Zod v4 (Zero Legado do v3)

O `golang-zod` nasce diretamente sobre a arquitetura do **Zod v4**, descartando métodos depreciados do v3:

### 1.1. Formatos de String são Top-Level (`z.Email()`, `z.UUID()`, etc.)
No Zod v3 usava-se `z.string().email()`. No Zod v4 isso é depreciado. Em Go:
* **Identificadores**: `z.UUID()`, `z.UUIDv4()`, `z.UUIDv7()`, `z.GUID()`, `z.NanoID()`, `z.CUID2()`, `z.ULID()`, `z.XID()`, `z.KSUID()`
* **Rede & Web**: `z.Email()`, `z.URL()`, `z.HTTPURL()`, `z.Hostname()`, `z.IPv4()`, `z.IPv6()`, `z.MAC()`, `z.CIDRv4()`, `z.CIDRv6()`
* **Codificação & Tokens**: `z.Base64()`, `z.Base64URL()`, `z.Hex()`, `z.JWT()`, `z.Emoji()`, `z.E164()`, `z.CreditCard()`, `z.IBAN()`, `z.CurrencyCode()`
* **Datas/Horas ISO (`z.ISO.*`)**: `z.ISO.Date()`, `z.ISO.Time()`, `z.ISO.DateTime()`, `z.ISO.Duration()`

Em `z.String()` ficam apenas restrições e transformações genéricas de texto:
* **Checks**: `.Min(n)`, `.Max(n)`, `.Length(n)`, `.NonEmpty()`, `.Regex(re)`, `.Includes(s)`, `.StartsWith(s)`, `.EndsWith(s)`, `.Lowercase()`, `.Uppercase()`
* **Transforms**: `.Trim()`, `.Normalize()`, `.ToLowerCase()`, `.ToUpperCase()`, `.Slugify()`

### 1.2. Tipos Numéricos são Top-Level (`z.Int()`, `z.Float64()`, etc.)
Construtores diretos alinhados aos tipos primitivos do Go:
* `z.Float64()`, `z.Float32()`, `z.Number()`
* `z.Int()`, `z.Int8()`, `z.Int16()`, `z.Int32()`, `z.Int64()`
* `z.Uint()`, `z.Uint8()`, `z.Uint16()`, `z.Uint32()`, `z.Uint64()`, `z.BigInt()`
* **Checks numéricos**: `.GT(v)`, `.GTE(v)` / `.Min(v)`, `.LT(v)`, `.LTE(v)` / `.Max(v)`, `.Positive()`, `.NonNegative()`, `.Negative()`, `.NonPositive()`, `.MultipleOf(v)`.

### 1.3. Objetos no Zod v4 (`z.Object`, `z.StrictObject`, `z.LooseObject`)
* `z.Object(z.Shape{...})`: faz `strip` de chaves desconhecidas por padrão.
* `z.StrictObject(z.Shape{...})`: rejeita chaves desconhecidas com `unrecognized_keys`.
* `z.LooseObject(z.Shape{...})`: preserva chaves desconhecidas (substitui o antigo `.passthrough()`).
* Extensão de schemas via `.Extend(z.Shape{...})` (substitui o antigo `.merge()`).

### 1.4. `z.Enum(...)` Idêntico ao Zod Original
Diferente do Rust (onde `enum` é palavra reservada da linguagem), em Go **`enum` não é palavra reservada**. Além disso, Go possui parâmetros variádicos nativos (`...T`), dispensando qualquer sufixo como `Slice`:

```go
// Uma única função variádica resolve tudo:
roleSchema := z.Enum("admin", "user", "guest")

// Se tiver um slice existente, usa o desempacotamento nativo do Go (...):
roles := []string{"admin", "user", "guest"}
roleSchema := z.Enum(roles...)

// Ou com tipos definidos (type Role string):
type Role string
const (
    Admin Role = "admin"
    User  Role = "user"
)
typedSchema := z.Enum(Admin, User)
```

---

## 2. Paradigmas Duais: Schema-First vs. Type-First com Generics

O Go 1.18+ trouxe Generics, permitindo que o `golang-zod` atenda com maestria a dois públicos:

### 2.1. Modo 1: Schema-First (Dinâmico / Tipagem Declarativa)
Ideal para validações pontuais, payloads flexíveis e parsing de APIs:
```go
var UserSchema = z.Object(z.Shape{
    "name":  z.String().Min(2).Trim(),
    "email": z.Email(),
    "age":   z.Int().Gte(18).Optional(),
})

// Uso com ParseInto estilo json.Unmarshal:
var user User
if issues := UserSchema.ParseInto(r.Body, &user); issues != nil {
    // Retorna erro estruturado
}
```

### 2.2. Modo 2: Type-First (`z.Object[T]()`) com Leitura de Tags
Elimina duplicidade de código inspecionando a struct **uma única vez** no startup:
```go
type User struct {
    FirstName string  `json:"first_name" zod:"min=2,trim"`
    Email     string  `json:"email"      zod:"email"`
    Bio       *string `json:"bio,omitempty"`          // omitempty ou ponteiro -> .Optional()
    Secret    string  `json:"-"`                      // json:"-" -> ignorado na validação
}

// Monta e compila a árvore AST uma única vez:
var UserSchema = z.Object[User]()

// Validação tipada direta:
user, err := UserSchema.Parse(r.Body) // 'user' já é do tipo User!
```

---

## 3. Arquitetura de Performance e Execução no Go

### 3.1. Zero Reflexão em Runtime (Bootstrap-only)
* A reflexão (`reflect`) é executada **exclusivamente** na chamada de inicialização `z.Object[T]()`.
* A struct é desmontada e convertida para nós compilados (`Node`) estáticos.
* Durante as requisições HTTP subsequentes, a validação é executada puramente através de chamadas de funções e ponteiros em memória.

### 3.2. Imutabilidade & Thread-Safety
* Todos os Schemas (`*Schema`, `*ObjectSchema`) são imutáveis após instanciados.
* Um único schema global pode ser compartilhado com segurança por centenas de goroutines simultâneas em servidores `net/http` / `fiber` / `gin` sem nenhum mutex ou race condition.

### 3.3. `sync.Pool` para Alocação Zero
* `ParseContext` e fatias de buffers de navegação (`path`) são reciclados via `sync.Pool`.
* Em requisições válidas (*happy path*), o custo de alocação de heap é mantido no menor patamar possível.

### 3.4. `AbortEarly`
* Possibilidade de interromper a validação no primeiro erro encontrado (`opt.AbortEarly = true`), poupando ciclos de CPU.

---

## 4. Taxonomia de Erros do Zod v4 e i18n

### 4.1. As 11 Issues Estruturadas (`IssueCode`)
1. `InvalidType`: tipo incompatível.
2. `TooBig`: tamanho/valor acima do máximo (`String`, `Number`, `Array`, `Map`, `Date`).
3. `TooSmall`: tamanho/valor abaixo do mínimo.
4. `InvalidFormat`: formato inválido (regex, email, uuid, datetime).
5. `NotMultipleOf`: divisor numérico inválido.
6. `UnrecognizedKeys`: chaves desconhecidas em strict objects.
7. `InvalidUnion`: nenhuma ou múltiplas variantes satisfeitas.
8. `InvalidKey`: chave de mapa ou elemento inválida.
9. `InvalidElement`: elemento de array ou set inválido.
10. `InvalidValue`: valor literal ou opção não permitida.
11. `Custom`: violação de regra customizada (`Refine`).

### 4.2. Os 4 Formatadores do Zod v4
```go
err.Format()   // Árvore aninhada com chave _errors
err.Flatten()  // Separação em FormErrors ([]string) e FieldErrors (map[string][]string)
err.Treeify()  // Novo no Zod v4: { errors, properties, items }
err.Prettify() // Formatação visual para terminal/logs com indicação em linha
```

### 4.3. i18n Nativo
* Suporte a troca global (`z.SetLocale(z.LocalePtBR)`) ou com escopo por requisição (`UserSchema.ParseWith(data, z.WithLocale(z.LocalePtBR))`).
* Mensagens internacionalizadas precisas para pluralização, gênero e unidades (`caracteres`, `elementos`, `bytes`).

---

## 5. Codecs Bidirecionais (`z.Codec`) e Transformações

Seguindo o Zod v4, o `golang-zod` suporta pipelines bidirecionais seguros:
* **Encode / Decode**:
  ```go
  base64Codec := z.Codec(z.String(), z.Bytes(), func(in string) ([]byte, error) {
      return base64.StdEncoding.DecodeString(in)
  }, func(out []byte) (string, error) {
      return base64.StdEncoding.EncodeToString(out), nil
  })
  ```
* **Coerção Explícita (`z.Coerce.*`)**: `z.Coerce.Int()`, `z.Coerce.Float()`, `z.Coerce.Boolean()`, `z.Coerce.Date()`.

---

## 6. O Que Foi Cortado do Zod (JS-ismos Desnecessários)

| Item no Zod TS | Status no `golang-zod` | Motivo |
| :--- | :--- | :--- |
| `.describe("...")` | **Cortado** | Substituído por `.Meta(z.Meta{Description: "..."})`. |
| `z.string().email()` | **Cortado** | Top-level no v4: `z.Email()`. |
| `z.number().int()` | **Cortado** | Top-level no v4: `z.Int()`. |
| `.passthrough()`, `.merge()` | **Cortado** | Substituídos por `.Loose()` / `z.LooseObject()` e `.Extend()`. |
| `z.enum([...])` | **Mantido como `z.Enum(...)`** | Em Go `enum` não é palavra reservada, e aceita variádico nativo `...T` (inclusive `slice...`). |
| `z.instanceof(...)` | **Cortado** | Específico de protótipos JS. |
| `z.undefined()`, `z.void()`, `z.symbol()` | **Cortado** | Tipos exclusivos do runtime JavaScript. |

---

## 7. Exportadores (JSON Schema & OpenAPI 3.1)

Capacidade nativa de introspecção da árvore AST:
1. **JSON Schema (Draft 2020-12 & Draft 07)**: `schema.ToJSONSchema(opts)`
2. **OpenAPI 3.1 Spec**: Exporta definições prontas para ferramentas como Swagger UI e Redoc.
3. **Importação Reversa**: `z.FromJSONSchema(jsonBytes)` para gerar schemas Go a partir de contratos externos em runtime.
