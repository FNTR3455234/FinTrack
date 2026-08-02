import { useCallback, useState } from 'react'

import tabla from '../componentes/Tabla.module.css'

import { AvisoDeError } from '../componentes/Aviso.jsx'
import Boton from '../componentes/Boton.jsx'
import ConfirmarBorrado from '../componentes/ConfirmarBorrado.jsx'
import Encabezado from '../componentes/Encabezado.jsx'
import { Cargando, Vacio } from '../componentes/Estados.jsx'
import Etiqueta, { TipoMovimiento } from '../componentes/Etiqueta.jsx'
import Tarjeta from '../componentes/Tarjeta.jsx'
import FormularioCategoria from './categorias/FormularioCategoria.jsx'
import { categorias as api } from '../api/catalogos.js'
import { usePeticion } from '../hooks/usePeticion.js'

export default function Categorias() {
  const [editando, setEditando] = useState(null)
  const [borrando, setBorrando] = useState(null)

  // Se piden tambien las archivadas: esta es la pantalla donde se administran,
  // y una categoria archivada que no se ve no se puede volver a activar.
  const cargar = useCallback(
    ({ signal }) => api.listar({ incluir_archivadas: true }, { signal }),
    [],
  )
  const { datos, cargando, error, recargar } = usePeticion(cargar)

  async function guardar(valores) {
    if (editando.id) {
      await api.actualizar(editando.id, valores)
    } else {
      await api.crear(valores)
    }
    recargar()
  }

  return (
    <>
      <Encabezado titulo="Categorias" descripcion="Como clasificas tus movimientos.">
        <Boton onClick={() => setEditando({})}>Nueva categoria</Boton>
      </Encabezado>

      {error && <AvisoDeError error={error} alReintentar={recargar} />}

      <Tarjeta>
        {cargando && !datos && <Cargando etiqueta="Cargando las categorias" filas={5} />}

        {datos && datos.length === 0 && (
          <Vacio
            titulo="Todavia no tienes categorias"
            accion={<Boton onClick={() => setEditando({})}>Crear la primera</Boton>}
          >
            Las categorias son lo que hace que los reportes digan algo: sin ellas todos los
            gastos son solo "gastos".
          </Vacio>
        )}

        {datos && datos.length > 0 && (
          <div className={tabla.envoltura}>
            <table className={tabla.tabla}>
              <thead>
                <tr>
                  <th scope="col">Categoria</th>
                  <th scope="col">Tipo</th>
                  <th scope="col">Estado</th>
                  <th scope="col"><span className="solo-lectores">Acciones</span></th>
                </tr>
              </thead>
              <tbody>
                {datos.map((categoria) => (
                  <tr key={categoria.id}>
                    <td className={tabla.principal}>
                      <span className={tabla.punto} style={{ background: categoria.color }} />
                      {/* El emoji es decorativo: el nombre ya dice de que es la
                          categoria, y leerlo en voz alta solo estorbaria. */}
                      <span aria-hidden="true">{categoria.icono} </span>
                      {categoria.nombre}
                    </td>
                    <td>
                      <TipoMovimiento tipo={categoria.tipo} />
                    </td>
                    <td>{categoria.archivada ? <Etiqueta>Archivada</Etiqueta> : <span className={tabla.tenue}>Activa</span>}</td>
                    <td>
                      <div className={tabla.acciones}>
                        <Boton variante="texto" onClick={() => setEditando(categoria)}>
                          Editar
                        </Boton>
                        <Boton variante="texto" onClick={() => setBorrando(categoria)}>
                          Borrar
                        </Boton>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Tarjeta>

      {editando && (
        <FormularioCategoria
          key={editando.id || 'nueva'}
          abierto
          categoria={editando.id ? editando : null}
          alGuardar={guardar}
          alCerrar={() => setEditando(null)}
        />
      )}

      {borrando && (
        <ConfirmarBorrado
          abierto
          nombre={borrando.nombre}
          alCerrar={() => setBorrando(null)}
          alConfirmar={() => api.eliminar(borrando.id).then(recargar)}
        />
      )}
    </>
  )
}
