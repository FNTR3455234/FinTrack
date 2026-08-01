package middleware

import (
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// ventana lleva la cuenta de peticiones de una IP en el periodo actual.
type ventana struct {
	conteo   int
	reinicio time.Time
}

// limitador es un contador de ventana fija, en memoria.
//
// Se eligio el algoritmo mas simple a proposito: para proteger /auth de fuerza
// bruta basta con cortar a quien intente demasiadas veces por minuto. Al ser en
// memoria, el limite es por instancia; con varias replicas habria que moverlo a
// Redis, pero eso no aplica al alcance de este proyecto.
type limitador struct {
	mu       sync.Mutex
	visitas  map[string]*ventana
	maximo   int
	duracion time.Duration
}

// LimitePeticiones corta con 429 a quien pase de maximo peticiones por ventana.
//
// Se aplica solo al grupo /auth: son los endpoints donde un atacante prueba
// contraseñas o inunda de registros.
func LimitePeticiones(maximo int, duracion time.Duration) gin.HandlerFunc {
	lim := &limitador{
		visitas:  make(map[string]*ventana),
		maximo:   maximo,
		duracion: duracion,
	}

	return func(c *gin.Context) {
		permitido, esperar := lim.registrar(c.ClientIP(), time.Now())
		if !permitido {
			// Retry-After le dice al cliente cuantos segundos esperar.
			c.Header("Retry-After", strconv.Itoa(int(esperar.Seconds())+1))
			respuestas.Fallo(c, errores.DemasiadasPeticiones(errores.CodigoDemasiadosIntentos,
				"Demasiados intentos. Espera un momento y vuelve a intentarlo."))
			return
		}
		c.Next()
	}
}

// registrar suma una peticion de esa IP y dice si se permite.
// Devuelve tambien cuanto falta para que la ventana se reinicie.
func (l *limitador) registrar(ip string, ahora time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	actual, existe := l.visitas[ip]
	if !existe || ahora.After(actual.reinicio) {
		// Ventana nueva para esta IP.
		l.limpiar(ahora)
		l.visitas[ip] = &ventana{conteo: 1, reinicio: ahora.Add(l.duracion)}
		return true, 0
	}

	actual.conteo++
	if actual.conteo > l.maximo {
		return false, actual.reinicio.Sub(ahora)
	}
	return true, 0
}

// limpiar borra las ventanas ya vencidas para que el mapa no crezca sin fin.
// Se llama al abrir una ventana nueva, que es justo cuando el mapa puede crecer.
func (l *limitador) limpiar(ahora time.Time) {
	for ip, v := range l.visitas {
		if ahora.After(v.reinicio) {
			delete(l.visitas, ip)
		}
	}
}
