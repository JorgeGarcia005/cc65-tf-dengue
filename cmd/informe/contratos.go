package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/api"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth"
)

// operacion es lo que el informe toma de una operación de docs/api/openapi.yaml.
type operacion struct {
	resumen   string
	peticion  string   // esquema del cuerpo ($ref), si tiene
	respuesta string   // esquema de la respuesta 2xx ($ref), si tiene
	codigos   []string // códigos documentados, en orden
}

var reRef = regexp.MustCompile(`\$ref: *"#/components/schemas/([A-Za-z]+)"`)

// leerOpenAPI recorre el YAML por indentación (la del archivo es fija:
// rutas a 2 espacios, métodos a 4, campos a 6, códigos a 8). No es un parser
// general de YAML: basta para este archivo, y TestContratosCubrenLasRutas
// falla si deja de reconocer alguna operación.
func leerOpenAPI(ruta string) (map[string]*operacion, error) {
	b, err := os.ReadFile(ruta)
	if err != nil {
		return nil, err
	}
	ops := map[string]*operacion{}
	var op *operacion
	ruta, bloque, codigo := "", "", ""
	enPaths := false
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		l := sc.Text()
		ind := len(l) - len(strings.TrimLeft(l, " "))
		t := strings.TrimSpace(l)
		switch {
		case l == "paths:":
			enPaths = true
			continue
		case !enPaths || t == "" || strings.HasPrefix(t, "#"):
			continue
		case ind == 0:
			enPaths = false
			continue
		case ind == 2 && strings.HasPrefix(t, "/"):
			ruta, op = strings.TrimSuffix(t, ":"), nil
		case ind == 4 && slices.Contains([]string{"get:", "post:", "put:", "patch:", "delete:"}, t):
			op = &operacion{}
			ops[strings.ToUpper(strings.TrimSuffix(t, ":"))+" "+ruta] = op
			bloque = ""
		case op == nil:
		case ind == 6:
			bloque, codigo = strings.SplitN(t, ":", 2)[0], ""
			if v, ok := strings.CutPrefix(t, "summary:"); ok {
				op.resumen = strings.TrimSpace(v)
			}
		case bloque == "responses" && ind == 8 && strings.HasPrefix(t, `"`):
			codigo = strings.Trim(strings.SplitN(t, ":", 2)[0], `"`)
			op.codigos = append(op.codigos, codigo)
		}
		if op == nil {
			continue
		}
		if m := reRef.FindStringSubmatch(t); m != nil {
			switch {
			case bloque == "requestBody" && op.peticion == "":
				op.peticion = m[1]
			case bloque == "responses" && strings.HasPrefix(codigo, "2") && op.respuesta == "":
				op.respuesta = m[1]
			}
		}
	}
	return ops, sc.Err()
}

func rolesDe(r api.Ruta) string {
	if r.Modo == api.Publica {
		return "público"
	}
	s := "admin"
	if slices.Contains(r.Roles, auth.RolAnalista) {
		s = "analista, admin"
	}
	if r.Modo == api.Ticket {
		s += " (ticket)"
	}
	return s
}

// regionContratos: una fila por ruta documentada de api.Rutas(), con el rol
// de la tabla de autorización y el contrato de docs/api/openapi.yaml.
func regionContratos() (string, error) {
	ops, err := leerOpenAPI("docs/api/openapi.yaml")
	if err != nil {
		return "", err
	}
	esquema := func(s string) string {
		if s == "" {
			return "—"
		}
		return "`" + s + "`"
	}
	var b strings.Builder
	b.WriteString("Tabla: Contratos de los endpoints de la API {#tab:contratos}\n\n")
	b.WriteString("| Método y ruta | Rol | Qué hace | Cuerpo de la petición | Respuesta correcta | Códigos documentados |\n")
	b.WriteString("|---|---|---|---|---|---|\n")
	for _, r := range api.Rutas() {
		if !r.Documentar {
			continue
		}
		clave := r.Metodo + " " + r.Ruta
		op, ok := ops[clave]
		if !ok {
			return "", fmt.Errorf("contratos: %s no está en docs/api/openapi.yaml", clave)
		}
		ruta := "`" + clave + "`"
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n", ruta, rolesDe(r), strings.TrimSuffix(op.resumen, "."),
			esquema(op.peticion), esquema(op.respuesta), strings.Join(op.codigos, ", "))
	}
	b.WriteString("\nNota: Rol y modo de autenticación tomados de `api.Rutas()`, la tabla de la que salen el enrutador y la autorización; resumen, esquemas y códigos, de `docs/api/openapi.yaml`. «Ticket»: el WebSocket se autentica con un ticket de un uso (sección 15.3). Los esquemas están en `components/schemas` del OpenAPI y en los `struct` de Go de la sección 12.2. Generado por `cmd/informe`.\n")
	return b.String(), nil
}

// regionPostman: los casos de docs/api/cc65-tf.postman_collection.json, con
// el cuerpo que envía cada uno y las aserciones (pm.test) que comprueba.
func regionPostman() (string, error) {
	b, err := os.ReadFile("docs/api/cc65-tf.postman_collection.json")
	if err != nil {
		return "", err
	}
	type item struct {
		Name    string `json:"name"`
		Request struct {
			Method string `json:"method"`
			URL    struct {
				Raw string `json:"raw"`
			} `json:"url"`
			Body struct {
				Raw string `json:"raw"`
			} `json:"body"`
		} `json:"request"`
		Event []struct {
			Listen string `json:"listen"`
			Script struct {
				Exec []string `json:"exec"`
			} `json:"script"`
		} `json:"event"`
	}
	var col struct {
		Item []item `json:"item"`
	}
	if err := json.Unmarshal(b, &col); err != nil {
		return "", err
	}
	reTest := regexp.MustCompile(`pm\.test\(\s*["'](.+?)["']`)
	var s strings.Builder
	s.WriteString("Tabla: Casos de prueba de la colección de Postman {#tab:postman}\n\n")
	s.WriteString("| # | Caso | Petición | Cuerpo | Aserciones |\n|---:|---|---|---|---|\n")
	for i, it := range col.Item {
		var tests []string
		for _, e := range it.Event {
			if e.Listen != "test" {
				continue
			}
			for _, m := range reTest.FindAllStringSubmatch(strings.Join(e.Script.Exec, "\n"), -1) {
				tests = append(tests, m[1])
			}
		}
		cuerpo := "—"
		if c := strings.TrimSpace(it.Request.Body.Raw); c != "" {
			var v any
			if json.Unmarshal([]byte(c), &v) == nil {
				compacto, _ := json.Marshal(v)
				c = string(compacto)
			} else { // con variables {{…}} no es JSON: solo se compacta
				c = strings.ReplaceAll(strings.Join(strings.Fields(c), " "), "{ ", "{")
				c = strings.ReplaceAll(c, " }", "}")
			}
			cuerpo = "`" + strings.ReplaceAll(c, "|", `\|`) + "`"
		}
		ruta := strings.TrimPrefix(it.Request.URL.Raw, "{{base_url}}")
		fmt.Fprintf(&s, "| %d | %s | `%s %s` | %s | %s |\n", i+1, it.Name, it.Request.Method, ruta, cuerpo,
			strings.ReplaceAll(strings.Join(tests, "; "), "|", `\|`))
	}
	s.WriteString("\nNota: Generado por `cmd/informe` desde `docs/api/cc65-tf.postman_collection.json`. Las variables entre llaves las completa la colección: el login guarda el token, el lanzamiento del entrenamiento guarda su identificador y el caso 5 elige al azar la semana y la edad (`{{semana_prueba}}`, `{{edad_prueba}}`), para que en cada corrida la primera predicción sea nueva y la repetida salga de la caché.\n")
	return s.String(), nil
}
