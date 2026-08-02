import { useState } from 'react'

import formulario from '../estilos/formulario.module.css'

import Aviso from '../componentes/Aviso.jsx'
import Boton from '../componentes/Boton.jsx'
import Campo from '../componentes/Campo.jsx'
import Encabezado from '../componentes/Encabezado.jsx'
import Tarjeta from '../componentes/Tarjeta.jsx'
import { useAuth } from '../contexto/AuthContexto.jsx'
import { useTema } from '../contexto/TemaContexto.jsx'
import { useAccion } from '../hooks/useAccion.js'
import { fechaCorta } from '../utiles/formato.js'

export default function Perfil() {
  const { usuario, actualizarPerfil, cerrarSesion } = useAuth()
  const { tema, alternar } = useTema()
  const [datos, setDatos] = useState({ nombre: usuario.nombre, moneda: usuario.moneda })
  const [guardado, setGuardado] = useState(false)
  const { ejecutar, ocupado, error } = useAccion(actualizarPerfil)

  function cambiar(campo) {
    return (evento) => {
      setGuardado(false)
      setDatos((actual) => ({ ...actual, [campo]: evento.target.value }))
    }
  }

  async function enviar(evento) {
    evento.preventDefault()
    const resultado = await ejecutar(datos)
    setGuardado(resultado.ok)
  }

  return (
    <>
      <Encabezado titulo="Perfil" descripcion="Tus datos y las preferencias de la aplicacion." />

      <div style={{ display: 'grid', gap: 16, maxWidth: 560 }}>
        <Tarjeta titulo="Datos de la cuenta">
          <form className={formulario.formulario} onSubmit={enviar} noValidate>
            {error && <Aviso tono="error">{error.texto}</Aviso>}
            {guardado && !error && <Aviso tono="exito">Los cambios se guardaron.</Aviso>}

            {/* El correo no se puede cambiar: es la credencial de acceso y la
                llave unica de la coleccion. Se muestra deshabilitado en vez de
                esconderlo, para que se vea con que cuenta se entro. */}
            <Campo etiqueta="Correo" type="email" value={usuario.email} disabled readOnly />

            <Campo
              etiqueta="Nombre"
              value={datos.nombre}
              onChange={cambiar('nombre')}
              minLength={2}
              maxLength={80}
              required
            />

            <Campo
              etiqueta="Moneda"
              value={datos.moneda}
              onChange={cambiar('moneda')}
              ayuda="Codigo de tres letras, por ejemplo MXN o USD."
              maxLength={3}
              required
            />

            <div className={formulario.acciones}>
              <Boton type="submit" ocupado={ocupado}>
                {ocupado ? 'Guardando' : 'Guardar cambios'}
              </Boton>
            </div>
          </form>
        </Tarjeta>

        <Tarjeta titulo="Preferencias">
          <div className={formulario.formulario}>
            <p style={{ fontSize: '0.88rem', color: 'var(--texto-tenue)' }}>
              Miembro desde {fechaCorta(usuario.fecha_registro)}. El tema se guarda en este
              navegador, no en tu cuenta.
            </p>
            <div className={formulario.acciones} style={{ justifyContent: 'space-between' }}>
              <Boton variante="secundario" onClick={alternar}>
                Cambiar a tema {tema === 'oscuro' ? 'claro' : 'oscuro'}
              </Boton>
              <Boton variante="peligro" onClick={cerrarSesion}>
                Cerrar sesion
              </Boton>
            </div>
          </div>
        </Tarjeta>
      </div>
    </>
  )
}
