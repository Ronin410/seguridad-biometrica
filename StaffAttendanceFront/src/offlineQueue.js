// Cola de marcajes manuales guardados en el dispositivo cuando la tablet se
// queda sin internet por completo (SPEC.md sección 8). Cada marcaje guarda
// su hora real de captura (capturado_en) para que, al sincronizar, el
// backend calcule puntualidad/retardo con esa hora y no con la hora en que
// se recuperó la conexión.
const STORAGE_KEY = 'staffattendance_cola_marcajes';

function leerCola() {
  try {
    return JSON.parse(localStorage.getItem(STORAGE_KEY)) || [];
  } catch {
    return [];
  }
}

function guardarCola(cola) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(cola));
  } catch {
    // almacenamiento lleno o bloqueado: el marcaje se pierde, pero no
    // debe tumbar la app.
  }
}

export function encolarMarcaje({ empleadoId, nombre, tipo }) {
  const cola = leerCola();
  cola.push({
    id: `${Date.now()}-${Math.random().toString(36).slice(2)}`,
    empleado_id: empleadoId,
    nombre,
    tipo,
    capturado_en: new Date().toISOString(),
  });
  guardarCola(cola);
}

export function contarPendientes() {
  return leerCola().length;
}

// Intenta enviar cada marcaje pendiente a /asistencia/marcar-manual. Los que
// se confirman se quitan de la cola; los que fallan por red se quedan para
// el siguiente intento.
export async function sincronizarCola(api) {
  const cola = leerCola();
  if (cola.length === 0) return { sincronizados: 0, pendientes: 0 };

  const restantes = [];
  let sincronizados = 0;

  for (const item of cola) {
    try {
      await api.post('/asistencia/marcar-manual', {
        empleado_id: item.empleado_id,
        tipo: item.tipo,
        capturado_en: item.capturado_en,
      });
      sincronizados++;
    } catch (err) {
      if (err.response) {
        // El servidor respondió (p. ej. duplicado o empleado inválido):
        // no tiene caso reintentarlo indefinidamente.
        sincronizados++;
      } else {
        restantes.push(item);
      }
    }
  }

  guardarCola(restantes);
  return { sincronizados, pendientes: restantes.length };
}
