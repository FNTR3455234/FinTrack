package modelos

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Estados de una meta de ahorro.
//
// El orden importa al decidir: primero se mira si ya se junto el dinero. Una
// meta que llego al objetivo esta cumplida aunque la fecha ya haya pasado; lo
// contrario seria decirle a alguien que fallo cuando en realidad lo consiguio.
const (
	EstadoMetaCumplida = "cumplida"
	EstadoMetaVencida  = "vencida"
	EstadoMetaEnCurso  = "en_curso"
)

// DiasPorMes es la longitud de mes que se usa para calcular el ritmo de ahorro.
//
// 30 dias fijos y no la longitud real: el ritmo es una orientacion ("unos 2 500
// al mes"), no una cifra contable, y usar meses de verdad haria que el mismo
// plan diera numeros distintos segun el mes en el que se mirara.
const DiasPorMes = 30

// ProgresoMeta es una meta con lo que se lleva ahorrado. Es la fila que
// devuelve la consulta relacional 3 (metas cruzadas con aportaciones), ya con
// los campos de calendario que le añade el servicio.
//
// El reparto no es casual: la base suma el dinero, que es donde estan los
// datos, y Go calcula lo que depende de la fecha de hoy, que es donde se puede
// probar con un reloj fijo.
type ProgresoMeta struct {
	MetaID        bson.ObjectID `bson:"meta_id"        json:"meta_id"`
	Nombre        string        `bson:"nombre"         json:"nombre"`
	Color         string        `bson:"color"          json:"color"`
	MontoObjetivo float64       `bson:"monto_objetivo" json:"monto_objetivo"`
	FechaLimite   time.Time     `bson:"fecha_limite"   json:"fecha_limite"`
	Notas         *string       `bson:"notas"          json:"notas"`
	Archivada     bool          `bson:"archivada"      json:"archivada"`

	// Lo que suma la agregacion.
	Ahorrado     float64    `bson:"ahorrado"      json:"ahorrado"`
	Restante     float64    `bson:"restante"      json:"restante"`
	Porcentaje   float64    `bson:"porcentaje"    json:"porcentaje"`
	Aportaciones int        `bson:"aportaciones"  json:"aportaciones"`
	UltimaFecha  *time.Time `bson:"ultima_fecha"  json:"ultima_fecha"`

	// Lo que añade el servicio a partir de la fecha de hoy.
	Estado        string  `bson:"-" json:"estado"`
	DiasRestantes int     `bson:"-" json:"dias_restantes"`
	RitmoMensual  float64 `bson:"-" json:"ritmo_mensual"`
}

// ResumenMetas son las cifras del conjunto, para encabezar la pantalla sin
// tener que recorrer la lista en el cliente.
type ResumenMetas struct {
	Total     int     `json:"total"`
	Cumplidas int     `json:"cumplidas"`
	Vencidas  int     `json:"vencidas"`
	Objetivo  float64 `json:"objetivo"`
	Ahorrado  float64 `json:"ahorrado"`
}

// MetaConAportaciones es lo que devuelve GET /metas/{id}: el progreso y el
// detalle de cada aportacion, que es lo que explica de donde sale la cifra.
type MetaConAportaciones struct {
	ProgresoMeta
	Detalle []Aportacion `json:"detalle"`
}

// ListadoMetas es lo que devuelve GET /metas.
//
// Va como objeto y no como arreglo suelto porque acompaña al resumen. El campo
// "meta" del sobre no sirve aqui: ese esta reservado para la paginacion y tiene
// forma fija.
type ListadoMetas struct {
	Metas   []ProgresoMeta `json:"metas"`
	Resumen ResumenMetas   `json:"resumen"`
}
