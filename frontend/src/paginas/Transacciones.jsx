import { useCallback, useMemo, useState } from 'react'

import estilos from './Transacciones.module.css'

import Aviso, { AvisoDeError } from '../componentes/Aviso.jsx'
import Boton from '../componentes/Boton.jsx'
import ConfirmarBorrado from '../componentes/ConfirmarBorrado.jsx'
import Encabezado from '../componentes/Encabezado.jsx'
import { Cargando, Vacio } from '../componentes/Estados.jsx'
import Paginacion from '../componentes/Paginacion.jsx'
import Tarjeta from '../componentes/Tarjeta.jsx'
import BarraCSV from './transacciones/BarraCSV.jsx'
import Filtros from './transacciones/Filtros.jsx'
import FormularioTransaccion from './transacciones/FormularioTransaccion.jsx'
import TablaMovimientos from './transacciones/TablaMovimientos.jsx'
import { categorias as apiCategorias, cuentas as apiCuentas } from '../api/catalogos.js'
import * as api from '../api/transacciones.js'
import { useAuth } from '../contexto/AuthContexto.jsx'
import { usePeticion } from '../hooks/usePeticion.js'
import { useRetardo } from '../hooks/useRetardo.js'

const FILTROS_VACIOS = {
  busqueda: '',
  desde: '',
  hasta: '',
  tipo: '',
  cuenta_id: '',
  categoria_id: '',
  orden: 'fecha_desc',
}

export default function Transacciones() {
  const { usuario } = useAuth()
  const [filtros, setFiltros] = useState(FILTROS_VACIOS)
  const [pagina, setPagina] = useState(1)
  const [editando, setEditando] = useState(null)
  const [borrando, setBorrando] = useState(null)
  const [alerta, setAlerta] = useState(null)

  // La busqueda espera a que se deje de escribir; el resto de los filtros se
  // aplican al instante porque son desplegables y fechas, que cambian de golpe.
  const busqueda = useRetardo(filtros.busqueda)
  const consulta = useMemo(
    () => sinVacios({ ...filtros, busqueda, pagina }),
    [filtros, busqueda, pagina],
  )

  // Los catalogos se piden una sola vez: alimentan los desplegables de los
  // filtros, los del formulario y la traduccion de identificadores a nombres en
  // la tabla.
  const cargarCatalogos = useCallback(
    ({ signal }) =>
      Promise.all([
        apiCuentas.listar(undefined, { signal }),
        apiCategorias.listar(undefined, { signal }),
      ]).then(([cuentas, categorias]) => ({ cuentas, categorias })),
    [],
  )
  const catalogos = usePeticion(cargarCatalogos)

  const cargarLista = useCallback(({ signal }) => api.listar(consulta, { signal }), [consulta])
  const lista = usePeticion(cargarLista, [JSON.stringify(consulta)])

  // guardar deja que el error suba: quien lo atrapa y lo enseña es el
  // formulario, que es donde el usuario acaba de escribir. Envolverlo aqui en un
  // useAccion haria que el modal se cerrara aunque el guardado fallara.
  async function guardar(valores) {
    if (editando.id) {
      await api.actualizar(editando.id, valores)
      setAlerta(null)
    } else {
      // Al crear un gasto, la API puede devolver una alerta si ese movimiento
      // deja su categoria cerca del limite del presupuesto o por encima. Se
      // guarda para enseñarla arriba, ya con el modal cerrado.
      const creada = await api.crear(valores)
      setAlerta(creada.alerta || null)
    }
    lista.recargar()
  }

  function cambiarFiltro(campo, valor) {
    setFiltros((actual) => ({ ...actual, [campo]: valor }))
    // Cualquier filtro nuevo devuelve a la primera pagina: quedarse en la
    // pagina 4 de un resultado que ahora tiene 2 mostraria una tabla vacia.
    setPagina(1)
  }

  const cuentasPorID = useMemo(() => porID(catalogos.datos?.cuentas), [catalogos.datos])
  const categoriasPorID = useMemo(() => porID(catalogos.datos?.categorias), [catalogos.datos])

  return (
    <>
      <Encabezado titulo="Movimientos" descripcion="Todo lo que entra y sale, con sus filtros.">
        <BarraCSV filtros={consulta} alImportar={lista.recargar} />
        <Boton onClick={() => setEditando({})} disabled={!catalogos.datos?.cuentas.length}>
          Nuevo movimiento
        </Boton>
      </Encabezado>

      {alerta && (
        <div className={estilos.alerta}>
          <Aviso
            tono={alerta.estado === 'excedido' ? 'error' : 'alerta'}
            titulo="Aviso de presupuesto"
            accion={
              <button type="button" className={estilos.cerrarAviso} onClick={() => setAlerta(null)}>
                Cerrar
              </button>
            }
          >
            {alerta.mensaje}
          </Aviso>
        </div>
      )}

      {catalogos.error && <AvisoDeError error={catalogos.error} alReintentar={catalogos.recargar} />}
      {lista.error && <AvisoDeError error={lista.error} alReintentar={lista.recargar} />}

      <div className={estilos.pila}>
        <Tarjeta>
          <Filtros
            valores={filtros}
            alCambiar={cambiarFiltro}
            alLimpiar={() => {
              setFiltros(FILTROS_VACIOS)
              setPagina(1)
            }}
            cuentas={catalogos.datos?.cuentas || []}
            categorias={catalogos.datos?.categorias || []}
          />
        </Tarjeta>

        <Tarjeta>
          {lista.cargando && !lista.datos && <Cargando etiqueta="Cargando los movimientos" filas={6} />}

          {lista.datos?.datos.length === 0 && (
            <Vacio
              titulo="Ningun movimiento con estos filtros"
              accion={<Boton onClick={() => setEditando({})}>Registrar uno</Boton>}
            >
              Prueba a quitar algun filtro o cambia el rango de fechas.
            </Vacio>
          )}

          {lista.datos?.datos.length > 0 && (
            <div data-cargando={lista.cargando || undefined} className={estilos.contenido}>
              <TablaMovimientos
                movimientos={lista.datos.datos}
                cuentas={cuentasPorID}
                categorias={categoriasPorID}
                moneda={usuario.moneda}
                alEditar={setEditando}
                alBorrar={setBorrando}
              />
              <Paginacion meta={lista.datos.meta} alCambiar={setPagina} />
            </div>
          )}
        </Tarjeta>
      </div>

      {editando && catalogos.datos && (
        <FormularioTransaccion
          key={editando.id || 'nuevo'}
          abierto
          transaccion={editando.id ? editando : null}
          cuentas={catalogos.datos.cuentas}
          categorias={catalogos.datos.categorias}
          alGuardar={guardar}
          alCerrar={() => setEditando(null)}
        />
      )}

      {borrando && (
        <ConfirmarBorrado
          abierto
          nombre={borrando.descripcion}
          alCerrar={() => setBorrando(null)}
          alConfirmar={() => api.eliminar(borrando.id).then(lista.recargar)}
        />
      )}
    </>
  )
}

// sinVacios quita los filtros que no se usaron. Mandar "tipo=" en la query
// funcionaria, pero deja una URL llena de ruido en la pestaña de red y en la
// bitacora del servidor.
function sinVacios(objeto) {
  return Object.fromEntries(Object.entries(objeto).filter(([, valor]) => valor !== ''))
}

// porID arma el indice de identificador a documento que usa la tabla.
function porID(lista) {
  return new Map((lista || []).map((elemento) => [elemento.id, elemento]))
}
