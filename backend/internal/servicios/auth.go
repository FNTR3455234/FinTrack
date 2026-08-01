// Package servicios contiene las reglas de negocio. Recibe los repositorios
// como interfaz, asi que se puede probar entero sin MongoDB.
package servicios

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"

	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/repositorios"
)

// RepositorioUsuarios es lo que el servicio necesita de la capa de datos.
// La interfaz se declara aqui, en quien la consume, no en el repositorio.
type RepositorioUsuarios interface {
	Crear(ctx context.Context, usuario *modelos.Usuario) error
	PorEmail(ctx context.Context, email string) (*modelos.Usuario, error)
	PorID(ctx context.Context, id bson.ObjectID) (*modelos.Usuario, error)
	Actualizar(ctx context.Context, id bson.ObjectID, nombre, moneda string) (*modelos.Usuario, error)
}

// Auth resuelve registro, inicio de sesion, refresco y perfil.
type Auth struct {
	usuarios RepositorioUsuarios
	tokens   *Tokens
}

// NuevoAuth arma el servicio con sus dependencias.
func NuevoAuth(usuarios RepositorioUsuarios, tokens *Tokens) *Auth {
	return &Auth{usuarios: usuarios, tokens: tokens}
}

// hashDeRelleno es un hash bcrypt valido de una contraseña cualquiera.
//
// Se compara contra el cuando el email no existe, para que el login tarde lo
// mismo exista o no el usuario. Sin esto, medir el tiempo de respuesta permite
// averiguar que correos estan registrados.
const hashDeRelleno = "$2a$10$76BgwHbDgsucfxkem2e6UO198KG2nLVp4uX6lFQ4WC/20ojmSigni"

// Registrar da de alta un usuario y le devuelve la sesion iniciada.
func (a *Auth) Registrar(ctx context.Context, peticion modelos.PeticionRegistro) (*modelos.RespuestaSesion, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(peticion.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errores.Interno(err)
	}

	moneda := strings.ToUpper(peticion.Moneda)
	if moneda == "" {
		moneda = modelos.MonedaPorDefecto
	}

	usuario := &modelos.Usuario{
		Nombre:        strings.TrimSpace(peticion.Nombre),
		Email:         normalizarEmail(peticion.Email),
		Password:      string(hash),
		Moneda:        moneda,
		FechaRegistro: time.Now().UTC(),
		Activo:        true,
	}

	if err := a.usuarios.Crear(ctx, usuario); err != nil {
		if errors.Is(err, repositorios.ErrDuplicado) {
			return nil, errores.Conflicto(errores.CodigoEmailYaRegistrado,
				"Ya existe una cuenta con ese correo.")
		}
		return nil, errores.Interno(err)
	}

	return a.sesionPara(usuario)
}

// IniciarSesion valida las credenciales y entrega los dos tokens.
func (a *Auth) IniciarSesion(ctx context.Context, peticion modelos.PeticionLogin) (*modelos.RespuestaSesion, error) {
	// Mismo error para "no existe el correo" y "la contraseña no es esa": si
	// se distinguieran, cualquiera podria averiguar quien tiene cuenta.
	credencialesInvalidas := errores.NoAutorizado(errores.CodigoCredencialesInvalidas,
		"El correo o la contraseña no son correctos.")

	usuario, err := a.usuarios.PorEmail(ctx, normalizarEmail(peticion.Email))
	if err != nil {
		if errors.Is(err, repositorios.ErrNoEncontrado) {
			// Se compara igual contra un hash de relleno para que el tiempo de
			// respuesta no delate que el correo no existe.
			_ = bcrypt.CompareHashAndPassword([]byte(hashDeRelleno), []byte(peticion.Password))
			return nil, credencialesInvalidas
		}
		return nil, errores.Interno(err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(usuario.Password), []byte(peticion.Password)); err != nil {
		return nil, credencialesInvalidas
	}

	if !usuario.Activo {
		return nil, errores.Prohibido(errores.CodigoCuentaDesactivada,
			"Esta cuenta esta desactivada.")
	}

	return a.sesionPara(usuario)
}

// Refrescar cambia un token de refresco vigente por uno de acceso nuevo.
func (a *Auth) Refrescar(ctx context.Context, peticion modelos.PeticionRefresco) (*modelos.RespuestaRefresco, error) {
	usuarioID, err := a.tokens.ValidarRefresco(peticion.TokenRefresco)
	if err != nil {
		if errors.Is(err, ErrTokenVencido) {
			return nil, errores.NoAutorizado(errores.CodigoTokenVencido,
				"Tu sesion expiro. Inicia sesion de nuevo.")
		}
		return nil, errores.NoAutorizado(errores.CodigoTokenInvalido,
			"El token de refresco no es valido.")
	}

	// Se vuelve a leer el usuario: pudo borrarse o desactivarse despues de que
	// se emitio el token de refresco, que dura siete dias.
	usuario, err := a.usuarios.PorID(ctx, usuarioID)
	if err != nil {
		if errors.Is(err, repositorios.ErrNoEncontrado) {
			return nil, errores.NoAutorizado(errores.CodigoTokenInvalido,
				"El token de refresco no es valido.")
		}
		return nil, errores.Interno(err)
	}
	if !usuario.Activo {
		return nil, errores.Prohibido(errores.CodigoCuentaDesactivada, "Esta cuenta esta desactivada.")
	}

	acceso, err := a.tokens.GenerarAcceso(usuario.ID)
	if err != nil {
		return nil, errores.Interno(err)
	}

	return &modelos.RespuestaRefresco{
		TokenAcceso: acceso,
		ExpiraEn:    a.tokens.SegundosDeAcceso(),
	}, nil
}

// sesionPara emite los dos tokens de un usuario recien autenticado.
func (a *Auth) sesionPara(usuario *modelos.Usuario) (*modelos.RespuestaSesion, error) {
	acceso, err := a.tokens.GenerarAcceso(usuario.ID)
	if err != nil {
		return nil, errores.Interno(err)
	}
	refresco, err := a.tokens.GenerarRefresco(usuario.ID)
	if err != nil {
		return nil, errores.Interno(err)
	}

	return &modelos.RespuestaSesion{
		TokenAcceso:   acceso,
		TokenRefresco: refresco,
		ExpiraEn:      a.tokens.SegundosDeAcceso(),
		Usuario:       *usuario,
	}, nil
}

// normalizarEmail deja el correo en minusculas y sin espacios, para que
// "Demo@Fintrack.mx " y "demo@fintrack.mx" sean el mismo usuario.
func normalizarEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
