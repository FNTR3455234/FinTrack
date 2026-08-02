import { useRef, useState } from 'react'

import estilos from './BarraCSV.module.css'

import Aviso from '../../componentes/Aviso.jsx'
import Boton from '../../componentes/Boton.jsx'
import Modal from '../../componentes/Modal.jsx'
import { exportarCSV, importarCSV } from '../../api/transacciones.js'
import { useAccion } from '../../hooks/useAccion.js'

// Los dos botones de CSV: exportar lo que se esta viendo e importar un archivo.
export default function BarraCSV({ filtros, alImportar }) {
  const entrada = useRef(null)
  const [resultado, setResultado] = useState(null)
  const exportacion = useAccion(descargar)
  const importacion = useAccion(importarCSV)

  // El archivo sale con los MISMOS filtros que la tabla. Si se exportara todo
  // siempre, el boton mentiria: quien acaba de filtrar julio espera julio.
  async function exportar() {
    await exportacion.ejecutar(filtros)
  }

  async function alElegirArchivo(evento) {
    const archivo = evento.target.files?.[0]
    // El input se limpia enseguida para que elegir el mismo archivo dos veces
    // seguidas vuelva a disparar el evento change.
    evento.target.value = ''
    if (!archivo) return

    const respuesta = await importacion.ejecutar(archivo)
    if (respuesta.ok) {
      setResultado(respuesta.datos)
      alImportar()
    }
  }

  return (
    <>
      <div className={estilos.barra}>
        <Boton variante="secundario" onClick={exportar} ocupado={exportacion.ocupado}>
          Exportar CSV
        </Boton>
        <Boton
          variante="secundario"
          onClick={() => entrada.current.click()}
          ocupado={importacion.ocupado}
        >
          Importar CSV
        </Boton>
        {/* El input de archivo se esconde y lo dispara el boton: el control
            nativo no se puede estilar y desentona con el resto. */}
        <input
          ref={entrada}
          type="file"
          accept=".csv,text/csv"
          className="solo-lectores"
          onChange={alElegirArchivo}
          aria-hidden="true"
          tabIndex={-1}
        />
      </div>

      {exportacion.error && <Aviso tono="error">{exportacion.error.texto}</Aviso>}

      <Modal
        abierto={Boolean(resultado)}
        titulo="Importacion terminada"
        alCerrar={() => setResultado(null)}
        pie={<Boton onClick={() => setResultado(null)}>Cerrar</Boton>}
      >
        <Aviso tono="exito">
          Se importaron {resultado?.importadas} movimientos.
        </Aviso>
      </Modal>

      <ErroresDeImportacion error={importacion.error} alCerrar={importacion.limpiar} />
    </>
  )
}

// ErroresDeImportacion enseña fila por fila lo que hay que arreglar.
//
// La API no importa nada si una sola fila falla, asi que este listado no es un
// resumen de lo que salio mal: es la lista completa de lo que hay que corregir
// antes de volver a subir el archivo.
function ErroresDeImportacion({ error, alCerrar }) {
  if (!error) return null

  return (
    <Modal
      abierto
      titulo="No se importo nada"
      alCerrar={alCerrar}
      pie={<Boton onClick={alCerrar}>Entendido</Boton>}
    >
      <Aviso tono="error">{error.message}</Aviso>

      {error.detalles.length > 0 && (
        <ul className={estilos.errores}>
          {error.detalles.map((detalle) => (
            <li key={detalle}>{detalle}</li>
          ))}
        </ul>
      )}

      <p className={estilos.nota}>
        El archivo se revisa entero antes de guardar nada. Corrige estas filas y vuelve a
        subirlo: asi no se cuela la mitad del archivo y no acabas con movimientos repetidos.
      </p>
    </Modal>
  )
}

// descargar pide el archivo y lo guarda.
//
// El navegador no deja escribir en el disco desde JavaScript: la unica forma es
// convertir la respuesta en una URL temporal y simular un clic sobre un enlace
// de descarga. La URL se libera despues para no dejar el blob en memoria.
async function descargar(filtros) {
  const { blob, nombre } = await exportarCSV(filtros)
  const url = URL.createObjectURL(blob)
  const enlace = document.createElement('a')
  enlace.href = url
  enlace.download = nombre
  document.body.appendChild(enlace)
  enlace.click()
  enlace.remove()
  URL.revokeObjectURL(url)
}
