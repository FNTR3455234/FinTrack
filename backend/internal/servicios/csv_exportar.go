package servicios

import (
	"context"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// Exportar devuelve las transacciones que cumplen el filtro como filas de
// texto, con el encabezado en la primera.
//
// Devuelve filas y no un archivo ya escrito para que la capa HTTP se encargue
// de codificar y de los encabezados de la respuesta, que es cosa suya. Y las
// arma en memoria, no sobre la conexion, porque un fallo a mitad de la escritura
// llegaria cuando ya se mando el 200 y el usuario se quedaria con un archivo
// truncado sin saberlo. El tope de MaximoFilasExportar acota esa memoria.
func (s *CSV) Exportar(ctx context.Context, usuarioID bson.ObjectID, filtro modelos.FiltroTransacciones) ([][]string, error) {
	cuentas, err := s.nombresDeCuentas(ctx, usuarioID)
	if err != nil {
		return nil, errores.Interno(err)
	}
	categorias, err := s.nombresDeCategorias(ctx, usuarioID)
	if err != nil {
		return nil, errores.Interno(err)
	}

	transacciones, err := s.transacciones.Todas(ctx, usuarioID, filtro)
	if err != nil {
		return nil, errores.Interno(err)
	}

	filas := make([][]string, 0, len(transacciones)+1)
	filas = append(filas, modelos.ColumnasCSV)

	for _, t := range transacciones {
		filas = append(filas, []string{
			t.Fecha.UTC().Format(time.DateOnly),
			t.Tipo,
			nombresPorID(cuentas, t.CuentaID),
			nombresPorID(categorias, t.CategoriaID),
			// 'f' con dos decimales: 850.5 se escribe "850.50", que es como se
			// lee una cantidad de dinero. El separador decimal es el punto, no
			// la coma, porque la coma ya separa las columnas.
			strconv.FormatFloat(t.Monto, 'f', 2, 64),
			t.Descripcion,
			notasComoTexto(t.Notas),
		})
	}
	return filas, nil
}

// notasComoTexto convierte el puntero a texto: null se escribe como celda vacia.
func notasComoTexto(notas *string) string {
	if notas == nil {
		return ""
	}
	return *notas
}

func (s *CSV) nombresDeCuentas(ctx context.Context, usuarioID bson.ObjectID) (map[bson.ObjectID]string, error) {
	cuentas, err := s.cuentas.Listar(ctx, usuarioID, true)
	if err != nil {
		return nil, err
	}

	nombres := make(map[bson.ObjectID]string, len(cuentas))
	for _, cuenta := range cuentas {
		nombres[cuenta.ID] = cuenta.Nombre
	}
	return nombres, nil
}

func (s *CSV) nombresDeCategorias(ctx context.Context, usuarioID bson.ObjectID) (map[bson.ObjectID]string, error) {
	categorias, err := s.categorias.Listar(ctx, usuarioID, "", true)
	if err != nil {
		return nil, err
	}

	nombres := make(map[bson.ObjectID]string, len(categorias))
	for _, categoria := range categorias {
		nombres[categoria.ID] = categoria.Nombre
	}
	return nombres, nil
}
