// Package config carga y valida la configuracion del servidor desde variables
// de entorno. Si falta algo o viene mal, el servidor no arranca: es preferible
// fallar al iniciar que descubrir el problema con la primera peticion.
package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Version de la API. La responde /health y se escribe en la bitacora al
// arrancar.
//
// Es var y no const para poder inyectarla al compilar, sin tocar el codigo:
//
//	go build -ldflags "-X github.com/FNTR3455234/FinTrack/backend/internal/config.Version=1.2.3"
//
// Asi la imagen de Docker lleva grabado el commit del que salio (ver
// backend/Dockerfile) y /health sirve para saber que version esta desplegada.
// El valor de aqui es el que se usa al compilar en local, sin la bandera.
var Version = "0.1.0-dev"

// Config son todos los valores que el servidor necesita para arrancar.
type Config struct {
	Puerto             string
	GinModo            string
	MongoURI           string
	MongoBD            string
	JWTSecretoAcceso   string
	JWTSecretoRefresco string
	JWTMinutosAcceso   int
	JWTDiasRefresco    int
	CORSOrigenes       []string
}

// Cargar lee el archivo .env (si existe) y despues el entorno del proceso.
// En Docker no hay .env y las variables llegan por el entorno, por eso el
// error de godotenv se ignora a proposito.
func Cargar() (*Config, error) {
	_ = godotenv.Load()
	return DesdeEntorno()
}

// DesdeEntorno construye la configuracion leyendo solo variables de entorno.
// Esta separada de Cargar para poder probarla sin tocar el sistema de archivos.
func DesdeEntorno() (*Config, error) {
	var fallos []string

	cfg := &Config{
		Puerto:             texto("PUERTO", "8080"),
		GinModo:            texto("GIN_MODO", "debug"),
		MongoURI:           texto("MONGO_URI", ""),
		MongoBD:            texto("MONGO_BD", "fintrack"),
		JWTSecretoAcceso:   texto("JWT_SECRETO_ACCESO", ""),
		JWTSecretoRefresco: texto("JWT_SECRETO_REFRESCO", ""),
		CORSOrigenes:       lista("CORS_ORIGENES", "http://localhost:5173"),
	}

	minutos, err := entero("JWT_MINUTOS_ACCESO", 15)
	if err != nil {
		fallos = append(fallos, err.Error())
	}
	cfg.JWTMinutosAcceso = minutos

	dias, err := entero("JWT_DIAS_REFRESCO", 7)
	if err != nil {
		fallos = append(fallos, err.Error())
	}
	cfg.JWTDiasRefresco = dias

	fallos = append(fallos, cfg.validar()...)
	if len(fallos) > 0 {
		// Se reportan todos los problemas juntos, no el primero: asi se corrige
		// el .env de una sola pasada.
		return nil, fmt.Errorf("configuracion invalida:\n  - %s", strings.Join(fallos, "\n  - "))
	}
	return cfg, nil
}

// validar devuelve la lista de problemas encontrados (vacia si todo esta bien).
func (c *Config) validar() []string {
	var fallos []string

	if _, err := strconv.Atoi(c.Puerto); err != nil {
		fallos = append(fallos, fmt.Sprintf("PUERTO debe ser un numero, se recibio %q", c.Puerto))
	}
	if c.GinModo != "debug" && c.GinModo != "release" && c.GinModo != "test" {
		fallos = append(fallos, fmt.Sprintf("GIN_MODO debe ser debug, release o test, se recibio %q", c.GinModo))
	}
	if c.MongoURI == "" {
		fallos = append(fallos, "MONGO_URI es obligatoria")
	}
	if c.MongoBD == "" {
		fallos = append(fallos, "MONGO_BD es obligatoria")
	}
	if c.JWTMinutosAcceso <= 0 {
		fallos = append(fallos, "JWT_MINUTOS_ACCESO debe ser mayor que 0")
	}
	if c.JWTDiasRefresco <= 0 {
		fallos = append(fallos, "JWT_DIAS_REFRESCO debe ser mayor que 0")
	}

	fallos = append(fallos, c.validarSecreto("JWT_SECRETO_ACCESO", c.JWTSecretoAcceso)...)
	fallos = append(fallos, c.validarSecreto("JWT_SECRETO_REFRESCO", c.JWTSecretoRefresco)...)

	if c.JWTSecretoAcceso != "" && c.JWTSecretoAcceso == c.JWTSecretoRefresco {
		fallos = append(fallos, "los secretos de acceso y de refresco deben ser distintos")
	}
	return fallos
}

// validarSecreto exige una longitud minima y, en produccion, que no se hayan
// dejado los valores de ejemplo del .env.example.
func (c *Config) validarSecreto(clave, valor string) []string {
	var fallos []string
	if valor == "" {
		return append(fallos, clave+" es obligatoria")
	}
	if len(valor) < 32 {
		fallos = append(fallos, fmt.Sprintf("%s debe tener al menos 32 caracteres (tiene %d)", clave, len(valor)))
	}
	if c.GinModo == "release" && strings.HasPrefix(valor, "cambia_este") {
		fallos = append(fallos, clave+" sigue con el valor de ejemplo y GIN_MODO es release")
	}
	return fallos
}

// EsProduccion indica si el servidor corre en modo release.
func (c *Config) EsProduccion() bool { return c.GinModo == "release" }

// Direccion es lo que espera http.Server en su campo Addr.
func (c *Config) Direccion() string { return ":" + c.Puerto }
