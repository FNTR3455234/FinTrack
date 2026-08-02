package servicios

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// marcaBOM son los tres bytes que Excel espera al principio de un UTF-8 y que
// la exportacion escribe. Hay que quitarlos al leer: si no, la primera columna
// del encabezado llegaria con esos bytes pegados delante de "fecha" y no se
// reconoceria. Es lo que hace que un archivo exportado se pueda volver a subir.
var marcaBOM = []byte{0xEF, 0xBB, 0xBF}

// Importar lee un CSV y guarda sus transacciones.
//
// O entra el archivo completo o no entra nada: primero se valida hasta la
// ultima fila y solo despues se inserta. MongoDB esta en modo standalone y no
// hay transacciones multidocumento (ver docs/decisiones.md, decision 003), asi
// que insertar sobre la marcha dejaria archivos importados por la mitad. Peor
// aun, reintentar duplicaria lo que si entro: en una app de dinero eso es un
// error que el usuario paga contando dos veces el mismo gasto.
func (s *CSV) Importar(ctx context.Context, usuarioID bson.ObjectID, lector io.Reader) (*modelos.ResultadoImportacion, error) {
	filas, err := leerFilas(lector)
	if err != nil {
		return nil, err
	}

	indices, err := indicesDeColumnas(filas[0])
	if err != nil {
		return nil, err
	}

	cuentas, err := s.catalogoDeCuentas(ctx, usuarioID)
	if err != nil {
		return nil, errores.Interno(err)
	}
	categorias, err := s.catalogoDeCategorias(ctx, usuarioID)
	if err != nil {
		return nil, errores.Interno(err)
	}

	transacciones, fallos := s.convertirFilas(usuarioID, filas[1:], indices, cuentas, categorias)
	if len(fallos) > 0 {
		return nil, errores.SolicitudInvalida(errores.CodigoCSVInvalido,
			"El archivo tiene filas que no se pueden importar. No se guardo ninguna.").
			ConDetalles(recortarErrores(fallos)...)
	}

	guardadas, err := s.transacciones.CrearVarias(ctx, transacciones)
	if err != nil {
		return nil, errores.Interno(err)
	}
	return &modelos.ResultadoImportacion{Importadas: guardadas}, nil
}

// leerFilas lee el archivo entero y comprueba lo que tiene que estar bien antes
// de mirar ni una sola celda.
func leerFilas(lector io.Reader) ([][]string, error) {
	datos, err := io.ReadAll(lector)
	if err != nil {
		return nil, errores.SolicitudInvalida(errores.CodigoCSVInvalido,
			"No se pudo leer el archivo.")
	}

	entrada := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(datos, marcaBOM)))
	entrada.TrimLeadingSpace = true

	filas, err := entrada.ReadAll()
	if err != nil {
		return nil, errores.SolicitudInvalida(errores.CodigoCSVInvalido,
			"El archivo no es un CSV valido.").ConDetalles(err.Error())
	}

	if len(filas) == 0 {
		return nil, errores.SolicitudInvalida(errores.CodigoCSVInvalido,
			"El archivo esta vacio: falta al menos la fila de encabezados.")
	}
	if len(filas)-1 > modelos.MaximoFilasImportar {
		return nil, errores.SolicitudInvalida(errores.CodigoCSVInvalido,
			fmt.Sprintf("El archivo trae %d filas y el maximo es %d.",
				len(filas)-1, modelos.MaximoFilasImportar))
	}
	return filas, nil
}

// indicesDeColumnas dice en que posicion viene cada columna.
//
// El orden del encabezado da igual y las mayusculas tampoco importan: el
// archivo lo puede haber armado una persona en Excel. Lo que si se exige es que
// esten todas las columnas.
func indicesDeColumnas(encabezado []string) (map[string]int, error) {
	posiciones := make(map[string]int, len(encabezado))
	for i, celda := range encabezado {
		posiciones[normalizarNombre(celda)] = i
	}

	faltantes := []string{}
	for _, columna := range modelos.ColumnasCSV {
		if _, existe := posiciones[columna]; !existe {
			faltantes = append(faltantes, columna)
		}
	}

	if len(faltantes) > 0 {
		return nil, errores.SolicitudInvalida(errores.CodigoCSVInvalido,
			"Al encabezado le faltan columnas.").
			ConDetalles("faltan: "+strings.Join(faltantes, ", "),
				"se esperaban: "+strings.Join(modelos.ColumnasCSV, ", "))
	}
	return posiciones, nil
}

// convertirFilas valida y convierte todas las filas, juntando TODOS los errores
// en vez de parar en el primero: quien sube un archivo con cinco erratas
// prefiere verlas de una vez y no descubrirlas de cinco en cinco intentos.
func (s *CSV) convertirFilas(usuarioID bson.ObjectID, filas [][]string, indices map[string]int, cuentas, categorias *catalogo) ([]modelos.Transaccion, []string) {
	transacciones := make([]modelos.Transaccion, 0, len(filas))
	fallos := []string{}
	momento := ahora()

	for i, fila := range filas {
		// +2: la fila 1 del archivo es el encabezado y las personas cuentan
		// desde 1, no desde 0. El numero tiene que coincidir con el que se ve
		// en la hoja de calculo.
		numero := i + 2

		if filaVacia(fila) {
			continue
		}

		transaccion, err := s.convertirFila(usuarioID, fila, indices, cuentas, categorias, momento)
		if err != nil {
			fallos = append(fallos, fmt.Sprintf("fila %d: %v", numero, err))
			continue
		}
		transacciones = append(transacciones, transaccion)
	}

	if len(transacciones) == 0 && len(fallos) == 0 {
		fallos = append(fallos, "el archivo no tiene ninguna fila con datos")
	}
	return transacciones, fallos
}

// filaVacia detecta las lineas en blanco que deja cualquier hoja de calculo al
// final del archivo.
func filaVacia(fila []string) bool {
	for _, celda := range fila {
		if strings.TrimSpace(celda) != "" {
			return false
		}
	}
	return true
}

// recortarErrores deja la lista de errores en un tamaño que se pueda leer.
func recortarErrores(fallos []string) []string {
	if len(fallos) <= modelos.MaximoErroresCSV {
		return fallos
	}
	recortados := append([]string{}, fallos[:modelos.MaximoErroresCSV]...)
	return append(recortados, fmt.Sprintf("y %d errores mas", len(fallos)-modelos.MaximoErroresCSV))
}
