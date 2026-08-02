package servicios

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// LectorCuentas y LectorCategorias son lo que la exportacion y la importacion
// necesitan para traducir entre nombres e identificadores.
//
// Se piden las archivadas tambien: un movimiento viejo puede apuntar a una
// cuenta que ya se archivo, y su nombre tiene que seguir apareciendo.
type LectorCuentas interface {
	Listar(ctx context.Context, usuarioID bson.ObjectID, incluirArchivadas bool) ([]modelos.Cuenta, error)
}

type LectorCategorias interface {
	Listar(ctx context.Context, usuarioID bson.ObjectID, tipo string, incluirArchivadas bool) ([]modelos.Categoria, error)
}

// RepositorioCSV es lo que el servicio necesita de la capa de datos.
type RepositorioCSV interface {
	Todas(ctx context.Context, usuarioID bson.ObjectID, filtro modelos.FiltroTransacciones) ([]modelos.Transaccion, error)
	CrearVarias(ctx context.Context, transacciones []modelos.Transaccion) (int, error)
}

// CSV exporta e importa transacciones en archivos separados por comas.
type CSV struct {
	transacciones RepositorioCSV
	cuentas       LectorCuentas
	categorias    LectorCategorias
}

// NuevoCSV arma el servicio con sus dependencias.
func NuevoCSV(t RepositorioCSV, c LectorCuentas, cat LectorCategorias) *CSV {
	return &CSV{transacciones: t, cuentas: c, categorias: cat}
}

// entradaCatalogo es lo que se sabe de una cuenta o una categoria por su nombre.
type entradaCatalogo struct {
	id   bson.ObjectID
	tipo string // solo lo usan las categorias
}

// catalogo traduce nombres a identificadores sin distinguir mayusculas ni
// espacios de sobra, que es como los escribe una persona en una hoja de calculo.
type catalogo struct {
	entradas  map[string]entradaCatalogo
	repetidos map[string]bool
	que       string
}

func nuevoCatalogo(que string) *catalogo {
	return &catalogo{
		entradas:  map[string]entradaCatalogo{},
		repetidos: map[string]bool{},
		que:       que,
	}
}

// agregar registra un nombre. Si ya estaba, lo marca como repetido en vez de
// pisarlo: con dos categorias que solo se distinguen por las mayusculas no hay
// forma de saber a cual se refiere una fila del archivo.
func (c *catalogo) agregar(nombre string, entrada entradaCatalogo) {
	llave := normalizarNombre(nombre)
	if _, existe := c.entradas[llave]; existe {
		c.repetidos[llave] = true
		return
	}
	c.entradas[llave] = entrada
}

// buscar resuelve un nombre del archivo.
func (c *catalogo) buscar(nombre string) (entradaCatalogo, error) {
	llave := normalizarNombre(nombre)

	if c.repetidos[llave] {
		return entradaCatalogo{}, fmt.Errorf("hay mas de una %s llamada %q y no se puede distinguir", c.que, nombre)
	}

	entrada, existe := c.entradas[llave]
	if !existe {
		return entradaCatalogo{}, fmt.Errorf("la %s %q no existe", c.que, nombre)
	}
	return entrada, nil
}

func normalizarNombre(nombre string) string {
	return strings.ToLower(strings.TrimSpace(nombre))
}

// catalogoDeCuentas arma el catalogo de cuentas del usuario.
func (s *CSV) catalogoDeCuentas(ctx context.Context, usuarioID bson.ObjectID) (*catalogo, error) {
	cuentas, err := s.cuentas.Listar(ctx, usuarioID, true)
	if err != nil {
		return nil, err
	}

	catalogo := nuevoCatalogo("cuenta")
	for _, cuenta := range cuentas {
		catalogo.agregar(cuenta.Nombre, entradaCatalogo{id: cuenta.ID})
	}
	return catalogo, nil
}

// catalogoDeCategorias arma el catalogo de categorias del usuario, con su tipo.
func (s *CSV) catalogoDeCategorias(ctx context.Context, usuarioID bson.ObjectID) (*catalogo, error) {
	categorias, err := s.categorias.Listar(ctx, usuarioID, "", true)
	if err != nil {
		return nil, err
	}

	catalogo := nuevoCatalogo("categoria")
	for _, categoria := range categorias {
		catalogo.agregar(categoria.Nombre, entradaCatalogo{id: categoria.ID, tipo: categoria.Tipo})
	}
	return catalogo, nil
}

// nombresPorID invierte un catalogo, para la exportacion.
func nombresPorID(nombres map[bson.ObjectID]string, id bson.ObjectID) string {
	if nombre, existe := nombres[id]; existe {
		return nombre
	}
	// No deberia pasar: no se puede borrar una cuenta o una categoria con
	// movimientos. Se deja explicito en vez de una celda vacia, que en una hoja
	// de calculo se confunde con un dato que falta.
	return "(sin nombre)"
}
