// Package apidoc embebe la especificación OpenAPI de la API para que el
// servidor la sirva en /docs sin depender de rutas del disco.
package apidoc

import _ "embed"

// OpenAPI es el contenido de docs/api/openapi.yaml.
//
//go:embed openapi.yaml
var OpenAPI []byte
