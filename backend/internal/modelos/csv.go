package modelos

// Columnas del CSV de transacciones, en este orden.
//
// La cuenta y la categoria van por NOMBRE y no por identificador: un CSV se
// abre en Excel y se edita a mano, y ahi un ObjectID de 24 caracteres no le
// dice nada a nadie. Como la importacion tambien resuelve por nombre, lo que
// exporta la API se puede volver a importar tal cual.
var ColumnasCSV = []string{
	"fecha", "tipo", "cuenta", "categoria", "monto", "descripcion", "notas",
}

// Limites de la exportacion y la importacion.
//
// Existen para que una peticion no pueda pedir toda la coleccion de golpe ni
// subir un archivo que no quepa en memoria. Con el volumen de una app de
// finanzas personales no se rozan: son un tope de seguridad, no una regla de
// negocio.
const (
	MaximoFilasExportar = 10000
	MaximoFilasImportar = 5000
	MaximoBytesCSV      = 2 << 20 // 2 MiB
	MaximoErroresCSV    = 20
)

// ResultadoImportacion es lo que responde POST /transacciones/importar cuando
// el archivo entero es valido.
//
// No hay importacion a medias: si una sola fila falla no se guarda ninguna, asi
// que Importadas siempre es igual al numero de filas del archivo.
type ResultadoImportacion struct {
	Importadas int `json:"importadas"`
}
