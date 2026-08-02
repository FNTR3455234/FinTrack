package servicios

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// convertirFila valida una fila del CSV y la convierte en una transaccion.
//
// Las reglas son EXACTAMENTE las mismas que las de POST /transacciones: mismo
// rango de monto, misma longitud de descripcion, mismo cruce de tipo entre el
// movimiento y su categoria. Una via de entrada que valide menos que la otra es
// una puerta trasera al modelo de datos.
//
// El dueño sale del token, igual que en el resto de la API: el archivo no puede
// decir de quien es cada movimiento.
func (s *CSV) convertirFila(usuarioID bson.ObjectID, fila []string, indices map[string]int, cuentas, categorias *catalogo, momento time.Time) (modelos.Transaccion, error) {
	celda := func(columna string) string {
		posicion := indices[columna]
		if posicion >= len(fila) {
			return ""
		}
		return strings.TrimSpace(fila[posicion])
	}

	fecha, err := time.Parse(time.DateOnly, celda("fecha"))
	if err != nil {
		return modelos.Transaccion{}, fmt.Errorf("la fecha %q no tiene el formato AAAA-MM-DD", celda("fecha"))
	}

	tipo := strings.ToLower(celda("tipo"))
	if tipo != modelos.TipoIngreso && tipo != modelos.TipoGasto {
		return modelos.Transaccion{}, fmt.Errorf("el tipo %q no es ingreso ni gasto", celda("tipo"))
	}

	monto, err := montoDeCelda(celda("monto"))
	if err != nil {
		return modelos.Transaccion{}, err
	}

	descripcion := celda("descripcion")
	if descripcion == "" {
		return modelos.Transaccion{}, fmt.Errorf("la descripcion es obligatoria")
	}
	if len([]rune(descripcion)) > 140 {
		return modelos.Transaccion{}, fmt.Errorf("la descripcion pasa de 140 caracteres")
	}

	notas := celda("notas")
	if len([]rune(notas)) > 500 {
		return modelos.Transaccion{}, fmt.Errorf("las notas pasan de 500 caracteres")
	}

	cuenta, err := cuentas.buscar(celda("cuenta"))
	if err != nil {
		return modelos.Transaccion{}, err
	}

	categoria, err := categorias.buscar(celda("categoria"))
	if err != nil {
		return modelos.Transaccion{}, err
	}
	if categoria.tipo != tipo {
		return modelos.Transaccion{}, fmt.Errorf("la categoria %q es de tipo %q y el movimiento dice %q",
			celda("categoria"), categoria.tipo, tipo)
	}

	return modelos.Transaccion{
		UsuarioID:     usuarioID,
		CuentaID:      cuenta.id,
		CategoriaID:   categoria.id,
		Tipo:          tipo,
		Monto:         monto,
		Descripcion:   descripcion,
		Fecha:         diaCalendario(fecha),
		Notas:         notasONulo(notas),
		CreadoEn:      momento,
		ActualizadoEn: momento,
	}, nil
}

// montoDeCelda convierte la cantidad y comprueba que tenga sentido.
//
// Se aceptan las comas de millar ("1,250.50") porque son lo que escribe Excel
// al dar formato de moneda, y el signo de pesos por lo mismo. Lo que NO se
// acepta es un monto negativo: el signo lo decide el tipo, no el numero.
func montoDeCelda(texto string) (float64, error) {
	limpio := strings.NewReplacer(",", "", "$", "", " ", "").Replace(texto)

	monto, err := strconv.ParseFloat(limpio, 64)
	if err != nil {
		return 0, fmt.Errorf("el monto %q no es un numero", texto)
	}
	if monto <= 0 {
		return 0, fmt.Errorf("el monto tiene que ser mayor que cero (el signo lo pone el tipo)")
	}
	return redondear(monto), nil
}
