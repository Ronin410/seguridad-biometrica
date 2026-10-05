package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

// negocioID lee el negocio_id que el middleware de auth deja en el contexto
// (viene del JWT). Todas las consultas de datos de negocio se filtran por
// este valor para aislar los datos entre negocios (SPEC.md sección 8).
func negocioID(c *gin.Context) int {
	value, _ := c.Get("negocio_id")
	id, _ := value.(int)
	return id
}

// pq's GenericArray only implements Scan for element types with an explicit
// sql.Scanner, so a smallint[] column can't be scanned directly into a plain
// []int — it must go through pq.Int64Array first. toIntSlice converts the
// result back to []int for the rest of the code.
func toIntSlice(values pq.Int64Array) []int {
	result := make([]int, len(values))
	for i, v := range values {
		result[i] = int(v)
	}
	return result
}
