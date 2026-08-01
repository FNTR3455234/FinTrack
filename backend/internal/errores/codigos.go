package errores

// Catalogo de codigos de error de la API.
//
// Son cadenas estables: el frontend decide que mensaje mostrar segun el codigo,
// no segun el texto del mensaje (que puede cambiar). Cada endpoint documenta en
// Swagger cuales puede devolver.
const (
	// Generales
	CodigoErrorInterno     = "ERROR_INTERNO"
	CodigoDatosInvalidos   = "DATOS_INVALIDOS"
	CodigoJSONInvalido     = "JSON_INVALIDO"
	CodigoIDInvalido       = "ID_INVALIDO"
	CodigoRutaNoEncontrada  = "RUTA_NO_ENCONTRADA"
	CodigoMetodoNoPermitido = "METODO_NO_PERMITIDO"
	CodigoBDNoDisponible   = "BD_NO_DISPONIBLE"

	// Autenticacion (fase 3)
	CodigoNoAutenticado         = "NO_AUTENTICADO"
	CodigoTokenInvalido         = "TOKEN_INVALIDO"
	CodigoTokenVencido          = "TOKEN_VENCIDO"
	CodigoCredencialesInvalidas = "CREDENCIALES_INVALIDAS"
	CodigoEmailYaRegistrado     = "EMAIL_YA_REGISTRADO"
	CodigoUsuarioNoEncontrado   = "USUARIO_NO_ENCONTRADO"
	CodigoCuentaDesactivada     = "CUENTA_DESACTIVADA"
	CodigoDemasiadosIntentos    = "DEMASIADOS_INTENTOS"

	// Recursos (fases 4 y 5)
	CodigoCuentaNoEncontrada      = "CUENTA_NO_ENCONTRADA"
	CodigoCategoriaNoEncontrada   = "CATEGORIA_NO_ENCONTRADA"
	CodigoTransaccionNoEncontrada = "TRANSACCION_NO_ENCONTRADA"
	CodigoPresupuestoNoEncontrado = "PRESUPUESTO_NO_ENCONTRADO"

	// Reglas de negocio (fases 4 y 5)
	CodigoCuentaConTransacciones    = "CUENTA_CON_TRANSACCIONES"
	CodigoCategoriaConTransacciones = "CATEGORIA_CON_TRANSACCIONES"
	CodigoPresupuestoDuplicado      = "PRESUPUESTO_DUPLICADO"
	CodigoTipoNoCoincide            = "TIPO_NO_COINCIDE"
)
