# INSIGHT.md — Go 1.27, Arquitetura e API do Zod v4 (`golang-zod`)

Fundação técnica do **`golang-zod`**, baseada no **Zod v4** (`colinhacks/zod`, `packages/zod/src/v4/`) e **validada contra o toolchain `go1.27.1`**. Cada decisão abaixo respeita o que o compilador Go aceita de fato.

Princípios:
* **Apenas API nativa do Zod v4** (zero legado do v3).
* **Zero tags mágicas de validação**: regras pertencem ao código Go (`z.String().Min(2)`), com autocomplete e checagem de tipos em compilação. Tags customizadas (`zod:"..."`) são proibidas.
* **A única tag aceita é `json:"..."`**, usada só por `z.Struct` para dar nome à chave de cada campo registrado.
* **Dois construtores de objeto, ambos com builder**: `z.Object(func(s *z.ObjectBuilder){...})` (chaves `string`, saída `map[string]any`) e `z.Struct(func(s *z.StructBuilder, t *T){...})` (ponteiros de campo, saída `T`). Chaves de struct vêm sempre de `json:"..."`.
* **Reflexão só na construção de `z.Struct`**; `unsafe` confinado a `struct_unsafe.go`. Não existe `ParseInto`.
* **Entrada por streaming** (`encoding/json/v2` + `jsontext`) para reduzir alocação.
* **Erros como valores & i18n nativo.**
* **Schemas imutáveis**, seguros para goroutines; `sync.Pool` para o contexto de parse.

---

## 0. Limites verificados no Go 1.27.1

Resultado de compilação real, que condiciona todo o design:

| Recurso                                                                                                                                | Resultado                                                 | Consequência                                                                                                           |
| :------------------------------------------------------------------------------------------------------------------------------------- | :-------------------------------------------------------- | :--------------------------------------------------------------------------------------------------------------------- |
| Métodos com type parameter (`func (s S) Map[T any](v T) T`)                                                                            | **Compila**                                               | `Transform[U]`, `Pipe[U]` podem ser métodos de tipos concretos.                                                        |
| Método genérico em interface (`Foo[T any]()`)                                                                                          | **Erro**: `interface method must have no type parameters` | A interface `Schema[T]` comum só pode ter o núcleo não genérico; combinadores genéricos não entram nela.               |
| `func (b Base[T]) Optional() Base[*T]`                                                                                                 | **Erro**: `instantiation cycle`                           | O método não pode devolver o **mesmo tipo base** com `T` alterado.                                                     |
| `func (b Base[T]) Optional() OptionalSchema[T]` (tipo wrapper distinto, `Parse` retorna `Option[T]`; `Nullable` retorna `Nullable[T]`) | **Compila**                                               | `.Optional()` fluente **existe**, retornando um wrapper. O wrapper não expõe novos `.Optional()`, o que evita o ciclo. |
| `z.Enum[T ~string](v ...T)` variádico                                                                                                  | **Compila**                                               | `z.Enum("a","b")`, `z.Enum(roles...)` e `z.Enum(Admin, User)` funcionam.                                               |
| `encoding/json/v2`                                                                                                                     | **Compila**, sem `GOEXPERIMENT`                           | Pode ser a base do parsing em streaming.                                                                               |

Regra de design resultante:
* **Checks que preservam `T`** (`Min`, `Max`, `Regex`, `GT`…) retornam o próprio tipo e encadeiam livremente.
* **Modificadores que alteram `T`** (`Optional`, `Nullable`, `Nullish`, `Default`, `Array`) são **exclusivamente métodos encadeados** (`z.String().Min(2).Optional()`), retornando **tipos wrapper distintos** (nunca `Base[*T]`). **Não existe `z.Optional(s)` como função livre.**
* `Union` e `Codec` (vários schemas de entrada) são funções livres; `Transform[U]` é método genérico de tipo concreto.

---

## 1. API Nativa do Zod v4 (zero legado do v3)

### 1.1. Formatos de string são top-level
No Zod v3 usava-se `z.string().email()`; no v4 isso é depreciado. Em Go:
* **Identificadores**: `z.UUID()`, `z.UUIDv4()`, `z.UUIDv7()`, `z.GUID()`, `z.NanoID()`, `z.CUID2()`, `z.ULID()`, `z.XID()`, `z.KSUID()`
* **Rede & Web**: `z.Email()`, `z.URL()`, `z.HTTPURL()`, `z.Hostname()`, `z.IPv4()`, `z.IPv6()`, `z.MAC()`, `z.CIDRv4()`, `z.CIDRv6()`
* **Codificação & Tokens**: `z.Base64()`, `z.Base64URL()`, `z.Hex()`, `z.JWT()`, `z.Emoji()`, `z.E164()`, `z.CreditCard()`, `z.IBAN()`, `z.CurrencyCode()`
* **Datas/Horas ISO**: `z.ISO.Date()`, `z.ISO.Time()`, `z.ISO.DateTime()`, `z.ISO.Duration()`

`z.String()` fica só com restrições e transformações genéricas:
* **Checks** (preservam `string`, encadeáveis): `.Min(n)`, `.Max(n)`, `.Length(n)`, `.NonEmpty()`, `.Regex(re)`, `.Includes(s)`, `.StartsWith(s)`, `.EndsWith(s)`, `.Lowercase()`, `.Uppercase()`
* **Transforms** (preservam `string`): `.Trim()`, `.Normalize()`, `.ToLowerCase()`, `.ToUpperCase()`, `.Slugify()`

### 1.2. Tipos numéricos
Construtores alinhados aos primitivos de Go, sobre uma constraint numérica (`~int | ~int8 | … | ~float64`):
* `z.Float64()`, `z.Float32()`, `z.Number()`
* `z.Int()`, `z.Int8()`, `z.Int16()`, `z.Int32()`, `z.Int64()`
* `z.Uint()`, `z.Uint8()`, `z.Uint16()`, `z.Uint32()`, `z.Uint64()`, `z.BigInt()` (→ `*big.Int`)
* **Checks**: `.GT(v)`, `.GTE(v)` / `.Min(v)`, `.LT(v)`, `.LTE(v)` / `.Max(v)`, `.Positive()`, `.NonNegative()`, `.Negative()`, `.NonPositive()`, `.MultipleOf(v)`.

**Convenção de nomes (única, em todo o projeto):** siglas em maiúsculas (`GT`, `GTE`, `LT`, `LTE`, `URL`, `UUID`).

**Precisão:** `Int64`, `Uint64` e `BigInt` nunca passam por `float64`. A entrada é lida como token/`json.Number`, para não perder precisão acima de 2^53.

### 1.3. Objetos (`z.Object` com `.Strict()` / `.Loose()`)
Um único construtor, que recebe uma **função builder** (§3). O tratamento de chaves desconhecidas é um modificador que preserva o tipo. **Os nomes seguem o Zod para maximizar a compatibilidade.**
* `z.Object(...)`: **`strip` é o padrão**, igual ao Zod (v3 e v4): chaves desconhecidas são removidas do resultado.
* `.Strict()`: rejeita chaves desconhecidas com `unrecognized_keys`.
* `.Loose()`: preserva chaves desconhecidas no resultado (só em `z.Object`).
* `.Strip()`: volta explicitamente ao padrão.
* `.Catchall(schema)`: valida cada chave desconhecida contra `schema` (só em `z.Object`).
* `.Extend(...)`, `.Partial(...)`, `.Required(...)`: só em `z.Object`; `.Pick(...)`, `.Omit(...)`, `.Keyof()`, `.Strict()`, `.Strip()`: em `z.Object` e `z.Struct`.
* `z.StrictObject(...)` e `z.LooseObject(...)` existem como no Zod v4: equivalem a `.Strict()` e `.Loose()`.

```go
user := z.Object(func(s *z.ObjectBuilder) {
    s.Field("name", z.String())
}).Strict()
```

### 1.4. `z.Enum(...)`
Em Go `enum` não é palavra reservada e há variádicos nativos:

```go
roleSchema := z.Enum("admin", "user", "guest")

roles := []string{"admin", "user", "guest"}
roleSchema = z.Enum(roles...)

type Role string
const (
    Admin Role = "admin"
    User  Role = "user"
)
typedSchema := z.Enum(Admin, User) // saída tipada: Role
```

Limitação: todos os valores têm o mesmo tipo `T ~string` (ou outro tipo comparável); enums mistos não são suportados.

---

## 2. Núcleo de tipos: o que a linguagem permite

### 2.1. Interface mínima + combinadores como funções livres

```go
// Núcleo: sem métodos genéricos (proibido em interface).
// O contexto é um parâmetro OPCIONAL no final (variádico; só o primeiro valor é usado).
// Usa o context.Context padrão da stdlib (sem tipo próprio).
type Schema[T any] interface {
    Parse(in any, ctx ...context.Context) (T, error)
    ParseJSON(data []byte, ctx ...context.Context) (T, error)
    Decode(r io.Reader, ctx ...context.Context) (T, error)
    GetMeta() Meta
}

// Modificadores que mudam T: wrappers distintos (evitam o instantiation cycle).
type OptionalSchema[T any] struct{ inner Schema[T] } // Parse -> Option[T]       (chave pode faltar)
type NullableSchema[T any] struct{ inner Schema[T] } // Parse -> Nullable[T]         (chave obrigatória, aceita null)
type NullishSchema[T any]  struct{ inner Schema[T] } // Parse -> Option[Nullable[T]] (os dois)

// Modificadores fluentes: cada schema concreto (ou Base[T]) expõe:
func (b Base[T]) Optional() OptionalSchema[T]
func (b Base[T]) Nullable() NullableSchema[T]
func (b Base[T]) Nullish() NullishSchema[T]
func (b Base[T]) Default(v T) Schema[T]
func (b Base[T]) Array() ArraySchema[T] // Parse -> []T

// Vários schemas de entrada: função livre.
func Union[T any](opts ...Schema[T]) Schema[T]

// Transform/Pipe: métodos genéricos em TIPOS CONCRETOS (permitidos no 1.27).
func (s *StringSchema) Transform[U any](f func(string) (U, error)) Schema[U]
```

Uso:
```go
age  := z.Int().GTE(18).Optional()
name := z.String().Min(2).Trim()
tags := z.String().NonEmpty().Array()
```

Limite: o wrapper não tem `.Optional()` próprio (`.Optional().Optional()` não compila, e não faz sentido). Quem precisar de `.Min()` após `.Optional()` deve aplicar o check **antes**.

### 2.2. Presença explícita: `Option[T]`
Go não distingue chave **ausente**, **`null`** e **valor zero** (`int` 0, `""`). Por isso:
* Dois tipos de saída distintos, porque são conceitos distintos (igual ao Zod):
  * `Option[T]` (`Some`/`None`): **ausência da chave**. Saída de `.Optional()`.
  * `Nullable[T]` (`Value`/`Null`): **valor `null`**. Saída de `.Nullable()`.
  * `.Nullish()` combina os dois: `Option[Nullable[T]]`.
* **Não se infere opcionalidade de `omitempty` nem de ponteiro.** Há uma única forma de declarar cada coisa: `.Optional()`, `.Nullable()`, `.Nullish()`.
* Detalhes de chaves de objeto em §2.2.1.

### 2.2.1. Optional e Nullable em chaves de objeto (caso principal)
* **`.Optional()`**: a chave **pode não existir**. Se existir, é validada normalmente. `null` **não** é aceito.
* **`.Nullable()`**: a chave **tem que existir** e aceita `null`. Ausente é erro.
* **`.Nullish()`**: a chave pode não existir **e** pode ser `null`.

| Entrada na chave    | Schema da chave            | Resultado                        |
| :------------------ | :------------------------- | :------------------------------- |
| ausente             | `Optional` / `Nullish`     | `None`, **sem erro**             |
| ausente             | `Nullable` ou schema comum | issue `InvalidType` (`required`) |
| `null`              | `Nullable` / `Nullish`     | `Null`, sem erro                 |
| `null`              | `Optional` ou schema comum | issue `InvalidType`              |
| presente (não nulo) | qualquer                   | validado normalmente             |

`.Default(v)` e `.Prefault(v)` só valem quando a chave está **ausente** (o `undefined` do Zod), **nunca para `null`**: `null` vai ao schema interno e falha, a menos que seja `.Nullable()`/`.Nullish()`. `.Default` devolve `v` sem revalidar; `.Prefault` passa `v` pelo schema interno.

```go
z.Object(func(s *z.ObjectBuilder) {
    s.Field("name",      z.String().Min(2))
    s.Field("bio",       z.String().Max(280).Optional())  // chave pode faltar; null não
    s.Field("deletedAt", z.ISO.DateTime().Nullable())     // chave obrigatória; aceita null
    s.Field("age",       z.Int().GTE(18).Nullish())       // pode faltar ou ser null
})
```
* Em `z.Object` (`map[string]any`), chave opcional ausente **não aparece**; `null` aparece como `nil`.
* Em `z.Struct`, o campo deve ser `z.Option[T]` para `.Optional()`, `z.Nullable[T]` para `.Nullable()` e `z.Option[z.Nullable[T]]` para `.Nullish()`; o compilador recusa o descasamento.

### 2.3. Assinaturas de entrada
`r.Body` é `io.Reader`; por isso há entradas distintas: `Parse(in)` para valores já decodificados, `ParseJSON([]byte)` e `Decode(io.Reader)` para streaming. Todas aceitam, **opcionalmente e no final**, um `context.Context` padrão: `schema.ParseJSON(body)` não precisa de nada, e `schema.ParseJSON(body, ctx)` usa o contexto. Quando passado, a biblioteca **o respeita**: o locale (§5.3) e também cancelamento e prazo (o parse para e devolve um erro que satisfaz `errors.Is(err, context.Canceled)` ou `context.DeadlineExceeded`, que não é um `*ZodError`). Sem contexto vale `context.Background()`: locale global, sem cancelamento. Não existe escopo implícito (o Go não tem armazenamento por goroutine): o contexto vale onde é passado. Não há tipo `ParseOption`: configurações por chamada viajam no próprio contexto, sob **chaves padrão documentadas**, strings com o caminho do módulo: `"github.com/leandroluk/golang-zod#locale"` (`z.LocaleContextKey`) e `"github.com/leandroluk/golang-zod#abortEarly"` (`z.AbortEarlyContextKey`). Qualquer código (outra biblioteca, um middleware) pode gravá-las com `context.WithValue` sem importar nada do `z`; `z.WithLocale(ctx, tag)` e `z.WithAbortEarly(ctx)` são só atalhos. (Um segundo variádico é impossível em Go, e `Option` é o tipo de presença `Option[T]`.) Observação: o `staticcheck` (SA1029) reclama de chave de contexto do tipo `string` no `context.WithValue`; é uma escolha consciente para a chave ser legível por qualquer código, sem tipo compartilhado.

---

## 3. Declaração de schemas e ligação com structs

Há **dois construtores**, ambos com **função builder** que roda **uma vez**, na criação do schema. Não há literal de mapa `z.ObjectBuilder{...}` nem `z.ObjectBy`. Não há DSL em string: toda regra é código Go checado pelo compilador.

```go
func Object(build func(s *Shape)) *ObjectSchema                          // chaves string, saída map[string]any (z.Dyn)
func Struct[T any](build func(s *StructBuilder, t *T)) *StructSchema[T]    // ponteiros de campo, saída T
func (s *Shape) Field[F any](name string, sch Schema[F])
func (s *StructBuilder) Field[F any](p *F, sch Schema[F])
```
Foram dois construtores porque, no go1.27.1, um literal `func(s *Shape)` **não** pode ser passado onde se espera `func(s *Shape, t *T)` (nem `func(s *Shape, t ...*T)`), e uma união dos dois tipos de função não infere `T`. Cada construtor tem tipos próprios, então as restrições são **de compilação**: em `StructBuilder.Field` um schema `.Optional()` sobre campo `string` não compila.

### 3.1. `z.Object`: saída dinâmica (`z.Dyn` = `map[string]any`)
```go
UserSchema := z.Object(func(s *z.ObjectBuilder) {
    s.Field("name",  z.String().Min(2).Trim())
    s.Field("email", z.Email())
    s.Field("age",   z.Int().GTE(18).Optional())
})

data, err := UserSchema.ParseJSON(body) // z.Dyn (map[string]any)
```

Não existe `ParseInto`: quem quer uma struct tipada usa `z.Struct` (§3.2), e `z.Object` devolve sempre `map[string]any`.

### 3.2. `z.Struct`: saída struct
Com ponteiros para os campos. O `build` recebe um `T` zerado:
```go
var UserSchema = z.Struct(func(s *z.StructBuilder, t *User) {
    s.Field(&t.Name,  z.String().Min(2))
    s.Field(&t.Email, z.Email())
    s.Field(&t.Bio,   z.String().Max(280).Optional()) // campo deve ser z.Option[string]
})

user, err := UserSchema.ParseJSON(body) // user é do tipo User
```
* **Identificação do campo:** `reflect` (só na construção) compara o endereço `&t.Foo` com os offsets dos campos de `T` e confere o tipo. O nome da chave vem da tag `json:"..."`, **obrigatória** em todo campo registrado (sem fallback para o nome do campo, KISS e o mais próximo do Zod, onde as chaves são sempre explícitas). `omitempty` não afeta a validação e nem ponteiros implicam opcionalidade.
* **Parse sem `reflect`:** a escrita nos campos usa `unsafe.Add` com o offset validado na construção, restrita a `struct_unsafe.go`.
* **Limites:** só campos exportados de nível superior (sem embutidos nem caminhos aninhados; struct aninhada é um `z.Struct[Inner]` próprio); `T` não struct, campo não exportado, sem tag `json` ou nome duplicado dão `panic` na construção.
* **Métodos:** `*StructSchema[T]` tem `.Strict()`, `.Strip()`, `.Pick()`, `.Omit()` e `.Keyof()`. **Não tem** `.Loose()`, `.Catchall()`, `.Extend()`, `.Partial()`, `.Required()` (uma struct não tem onde guardar chaves extras nem muda de campos): como são tipos distintos, o uso errado nem compila.
* **Por que `s.Field(&t.Foo, schema)` e não `s.Field(&t.Foo).String().Optional()`:** na forma encadeada, nada registra o resultado final da cadeia, e o compilador não consegue recusar `.String()` num campo `Option[string]`.

---

## 4. Performance e execução

### 4.1. Reflexão: onde é usada
* `z.Struct`: `reflect` só na construção; no parse, escrita por `unsafe.Add` (sem `reflect`).
* `Parse`, `ParseJSON` e `Decode` nunca usam `reflect`.

### 4.2. Streaming de entrada
* `ParseJSON`/`Decode` leem tokens com `encoding/json/v2` + `jsontext.Decoder`, validando e construindo a saída (`map[string]any`) em uma única passada, sem materializar um `any` intermediário.
* Preserva precisão numérica.
* Alocação mínima no happy path é uma **meta** a ser medida com benchmarks (`-benchmem`), não uma garantia de projeto.

### 4.3. Imutabilidade & thread-safety
Schemas são imutáveis após construídos; um schema global pode ser compartilhado por muitas goroutines sem mutex.

### 4.4. `sync.Pool`
`ParseContext` e buffers de caminho (`path`) são reciclados via `sync.Pool`.

### 4.5. `AbortEarly`
`ctx = z.WithAbortEarly(ctx)` (ou `context.WithValue(ctx, "github.com/leandroluk/golang-zod#abortEarly", true)`) interrompe no primeiro erro encontrado; passe `ctx` ao parse (`schema.ParseJSON(body, ctx)`).

---

## 5. Taxonomia de erros e i18n

### 5.1. As 11 issues (`IssueCode`)
1. `InvalidType` 2. `TooBig` 3. `TooSmall` 4. `InvalidFormat` 5. `NotMultipleOf` 6. `UnrecognizedKeys` 7. `InvalidUnion` 8. `InvalidKey` 9. `InvalidElement` 10. `InvalidValue` 11. `Custom`

### 5.2. Formatadores
```go
err.Flatten()  // FormErrors []string + FieldErrors map[string][]string (chave = só o PRIMEIRO segmento do caminho)
err.Treeify()  // { errors, properties, items }
err.Prettify() // "✖ mensagem" + "  → at caminho", ordenado pelo tamanho do caminho
```

Campos das issues (conferidos no `core/errors.ts` do Zod v4): `invalid_type` tem só `expected` (não existe `received`); `too_big`/`too_small` têm `origin`, `maximum`/`minimum`, `inclusive`, `exact`; `invalid_format` tem `format` e, conforme o formato, `pattern`, `prefix`, `suffix`, `includes`, `algorithm`; `invalid_union` tem `errors`, `discriminator`, `options`, `inclusive` e `matches`; `invalid_key`/`invalid_element` têm `origin` e `issues` (e `key` no segundo). A lista completa está no spec 01 (REQ-CORE-006).

### 5.3. i18n
* Padrão global: `z.SetLocale(z.LocalePtBR)`.
* **Escopo por requisição, no `context.Context`, com uma função só**: `ctx = z.WithLocale(ctx, "pt-BR")`, e `schema.ParseJSON(data, ctx)` (o contexto é opcional no parse; sem ele vale o locale global). O segundo argumento aceita uma tag ou o valor cru do header `Accept-Language`; vazio ou desconhecido devolve o `ctx` sem mudança. O schema continua imutável e compartilhado. **Chave padrão no contexto:** `"github.com/leandroluk/golang-zod#locale"` (`z.LocaleContextKey`): `z.WithLocale` só grava nela, e qualquer código pode fazer o mesmo: `context.WithValue(ctx, "github.com/leandroluk/golang-zod#locale", "pt-BR")` (valor `Locale` ou `string`).
* **Configuração global (opcional): trocar as chaves padrão.** `z.Config.UseContext(z.ContextOptions{LocaleKey: "lang"})` no `main` **substitui** a chave `"github.com/leandroluk/golang-zod#locale"` por `"lang"`, para compartilhar uma chave que a aplicação já usa. Depois disso a biblioteca lê `ctx.Value("lang")` e `z.WithLocale` grava em `"lang"`; a chave padrão antiga deixa de ser consultada. Campo vazio mantém o padrão daquela chave (`AbortEarlyKey` funciona igual). Valor do locale: `Locale` ou `string` com tag/`Accept-Language`. As chaves são `any`, pois chaves de contexto idiomáticas em Go são tipos próprios não exportados; strings também funcionam. `UseContext(ContextOptions{})` restaura os padrões.
* **Ordem de resolução:** `ctx.Value(<chave ativa>)` (a padrão, ou a que `UseContext` definiu) > `SetLocale` global > `LocaleEN`. As mensagens são resolvidas ao criar a issue, então o `*ZodError` já traz o texto na língua da requisição. `.Error(msg)` customizado vence qualquer locale.
* `z.RegisterLocale(tag, l)` registra locales customizados (`en`, `pt-BR` e `es` já vêm registrados).
* **Por que contexto e não um escopo por callback** (`z.WithLocale(tag, func() error {...})`): o Go não tem armazenamento por goroutine, então um `Parse` chamado dentro do callback não teria como achar o valor. O contexto é o parâmetro opcional no final de todo parse.
* Pluralização e unidades (`caracteres`, `elementos`, `bytes`) por locale.

#### Exemplo completo de uso
O contexto é um **parâmetro opcional no final**: `schema.ParseJSON(body)` não precisa de nada; passe um contexto só onde o idioma ou o cancelamento importam: `schema.ParseJSON(body, ctx)`. Não há escopo implícito: o middleware coloca o idioma no contexto da requisição e o handler o passa ao parse quando quiser aplicá-lo.

```go
// 1. main: opcional. Idioma padrão e, se a aplicação já guarda o idioma no contexto, troca da chave.
func main() {
    z.SetLocale(z.LocaleEN)
    // opcional: SUBSTITUI a chave padrão "github.com/leandroluk/golang-zod#locale" por "lang"
    // (a biblioteca passa a ler e a gravar "lang").
    z.Config.UseContext(z.ContextOptions{LocaleKey: "lang"})
    // ...
}

// 2. Middleware do Fiber: roda uma vez por requisição, sem limpeza.
app.Use(func(c *fiber.Ctx) error {
    // Header vazio ou desconhecido: WithLocale devolve o ctx sem mudança.
    c.SetUserContext(z.WithLocale(c.UserContext(), c.Get("Accept-Language")))
    return c.Next()
})

// 3. Handler: o contexto é opcional. Sem ele, vale o locale global e não há cancelamento:
//        user, err := UserSchema.ParseJSON(c.Body())
//    Com ele, mensagens na língua da requisição e o parse para se o cliente
//    desconectar ou o prazo estourar.
func createUser(c *fiber.Ctx) error {
    ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
    defer cancel()

    user, err := UserSchema.ParseJSON(c.Body(), ctx)
    if err != nil {
        var zerr *z.ZodError
        switch {
        case errors.As(err, &zerr):
            return c.Status(fiber.StatusUnprocessableEntity).JSON(zerr.Flatten()) // textos localizados
        case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
            return c.SendStatus(fiber.StatusRequestTimeout)
        default:
            return err
        }
    }
    _ = user // do tipo User quando UserSchema é um z.Struct
    return c.SendStatus(fiber.StatusCreated)
}

// 4. Sem HTTP: worker, CLI ou teste.
ctx := z.WithLocale(context.Background(), "pt-BR")
_, err := OrderSchema.ParseJSON(payload, ctx)

// 5. A aplicação já guarda o idioma no contexto sob a própria chave. O passo 1 TROCOU a chave
//    padrão por "lang", então a biblioteca lê "lang": dispensa z.WithLocale.
ctx = context.WithValue(ctx, "lang", "es")
_, err = OrderSchema.ParseJSON(payload, ctx) // mensagens em espanhol

// 5b. Sem o passo 1 (chave padrão), qualquer código pode gravar a chave documentada,
//     sem importar nada do z:
ctx = context.WithValue(ctx, "github.com/leandroluk/golang-zod#locale", "pt-BR")

// 6. Parar no primeiro erro (a configuração também viaja no contexto).
_, err = OrderSchema.ParseJSON(payload, z.WithAbortEarly(ctx))

// 7. Sem contexto algum: locale global, sem cancelamento.
_, err = OrderSchema.ParseJSON(payload)
```

---

## 6. Codecs e coerção

`Codec[A, B]` é função livre genérica (sem método genérico em interface):
```go
base64Codec := z.Codec(z.String(), z.Bytes(),
    func(in string) ([]byte, error) { return base64.StdEncoding.DecodeString(in) },
    func(out []byte) (string, error) { return base64.StdEncoding.EncodeToString(out), nil },
)
```
**Coerção explícita:** `z.Coerce.Int()`, `z.Coerce.Float()`, `z.Coerce.Boolean()`, `z.Coerce.Date()`.

---

## 7. O que foi cortado do Zod

| Item no Zod TS                               | Status                                                                        | Motivo                                                                                     |
| :------------------------------------------- | :---------------------------------------------------------------------------- | :----------------------------------------------------------------------------------------- |
| `.describe("...")`                           | Vira método do builder de metadados                                           | `z.String().Meta(func(m *z.MetaBuilder){ m.Describe("...") })`.                            |
| `formatError` (`Format()`)                   | Cortado                                                                       | Deprecated no Zod 4; use `Treeify`.                                                        |
| `.unique()` em arrays                        | Não existe                                                                    | Não está no Zod 4.                                                                         |
| `z.string().email()`                         | Cortado                                                                       | Top-level: `z.Email()`.                                                                    |
| `z.number().int()`                           | Cortado                                                                       | Top-level: `z.Int()`.                                                                      |
| `.merge()`                                   | Cortado                                                                       | Use `.Extend()`.                                                                           |
| `.passthrough()`                             | Cortado                                                                       | Deprecated no Zod 4; o nome do v4 é `.Loose()` (regra: só nomes do Zod v4 não deprecated). |
| `z.strictObject()`, `z.looseObject()`        | **Mantidos** (`z.StrictObject()`, `z.LooseObject()`)                          | Existem no Zod v4; coexistem com `.Strict()` / `.Loose()`.                                 |
| `.optional()` / `.nullable()` / `.nullish()` | **Mantidos** (somente encadeados: `.Optional()`, `.Nullable()`, `.Nullish()`) | Retornam tipos wrapper distintos, nunca `Base[*T]` (instantiation cycle, §0).              |
| Tags `zod:"..."`                             | **Proibidas**                                                                 | Regras ficam em código Go; só `json:"..."` é lida.                                         |
| `z.enum([...])`                              | Mantido como `z.Enum(...)`                                                    | Variádico nativo.                                                                          |
| `z.instanceof(...)`                          | Cortado                                                                       | Específico de protótipos JS.                                                               |
| `z.undefined()`, `z.void()`, `z.symbol()`    | Cortado                                                                       | Exclusivos do runtime JS.                                                                  |

---

## 8. Exportadores

1. **JSON Schema** (Draft 2020-12 padrão, Draft 07, Draft 04) e **OpenAPI 3.0**: `z.ToJSONSchema(schema, opts)`, com as opções do Zod: `io` (padrão `output`), `unrepresentable` (padrão `throw`; `any` vira `{}`), `cycles` (`ref`/`throw`), `reused` (`inline`/`ref`), `override`, `metadata`, `uri`. No Zod são **não representáveis** (erro por padrão): `bigint`, `int64`, `symbol`, `undefined`, `void`, `date`, `map`, `set`, `transform`, `nan`, `custom`. Objeto default emite `additionalProperties: false` em `output` e omite em `input`.
2. **OpenAPI 3.1**: alvo adicional, extensão que não existe no Zod.
3. **Importação reversa:** `z.FromJSONSchema(jsonBytes)` gera um schema dinâmico (`z.Object`) em runtime. No Zod é **experimental** e fora da API estável.

Como `Schema[T]` não pode ter métodos genéricos e a exportação não depende de `T`, o introspector opera sobre uma interface não genérica `Node` (AST), que todo schema expõe.

---

## 9. Pontos em aberto (validar com protótipo)

* Ergonomia de `Option[T]` frente a `*T` no dia a dia.
* Se checks após `.Optional()` (por exemplo `.Optional().Min(2)`) valem um custo extra, replicando os checks no wrapper, ou se a regra "check antes de Optional" basta.
* Custo real de alocação do streaming (`jsontext`) contra a abordagem via `map[string]any`.
* Divisão do pacote: `z` (API pública) e `internal/ast` (nós e introspecção).
