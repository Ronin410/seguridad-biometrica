import React, { useEffect, useRef, useState, useCallback } from 'react';
import Webcam from 'react-webcam';
import { ScanEye, RefreshCw, CheckCircle, AlertCircle, Wrench, CloudOff, CloudUpload } from 'lucide-react';
import api from '../axiosConfig';
import { encolarMarcaje, contarPendientes, sincronizarCola } from '../offlineQueue';

export default function Kiosco() {
  const webcamRef = useRef(null);
  const [loading, setLoading] = useState(false);
  const [resultado, setResultado] = useState(null);
  const [hoy, setHoy] = useState([]);
  const [mostrarManual, setMostrarManual] = useState(false);
  const [pendientes, setPendientes] = useState(() => contarPendientes());
  const [sincronizando, setSincronizando] = useState(false);

  const cargarHoy = async () => {
    try {
      const res = await api.get('/asistencia/hoy');
      setHoy(res.data || []);
    } catch (err) {
      console.error(err);
    }
  };

  const sincronizar = useCallback(async () => {
    if (contarPendientes() === 0) return;
    setSincronizando(true);
    try {
      const { sincronizados } = await sincronizarCola(api);
      setPendientes(contarPendientes());
      if (sincronizados > 0) cargarHoy();
    } finally {
      setSincronizando(false);
    }
  }, []);

  useEffect(() => {
    cargarHoy();
    sincronizar();
    window.addEventListener('online', sincronizar);
    return () => window.removeEventListener('online', sincronizar);
  }, [sincronizar]);

  const escanear = async () => {
    if (!webcamRef.current) return;
    const imageSrc = webcamRef.current.getScreenshot();
    if (!imageSrc) return;

    setLoading(true);
    setResultado(null);
    try {
      const res = await api.post('/asistencia/marcar', { imagen: imageSrc.split(',')[1] });
      setResultado({ type: 'success', data: res.data });
      cargarHoy();
    } catch (err) {
      setResultado({ type: 'error', msg: err.response?.data?.error || 'No se pudo procesar el marcaje' });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <div className="flex flex-col items-center gap-6">
        <div className="relative rounded-[3rem] overflow-hidden border-8 border-white bg-slate-200 shadow-2xl aspect-[3/4] w-full max-w-sm">
          <Webcam
            audio={false}
            ref={webcamRef}
            screenshotFormat="image/jpeg"
            videoConstraints={{ facingMode: 'user' }}
            className="absolute inset-0 w-full h-full object-cover"
            mirrored
          />
          {loading && (
            <div className="absolute inset-0 bg-white/60 backdrop-blur-sm flex items-center justify-center z-20">
              <RefreshCw className="animate-spin text-teal-700" size={48} />
            </div>
          )}
        </div>

        <button
          onClick={escanear}
          disabled={loading}
          className="w-full max-w-sm py-5 bg-teal-700 hover:bg-teal-800 text-white rounded-[1.5rem] font-black uppercase text-lg shadow-lg active:scale-95 transition-all disabled:opacity-50"
        >
          {loading ? 'Procesando...' : 'Marcar entrada / salida'}
        </button>

        <div className="flex items-center gap-4">
          <button
            onClick={() => setMostrarManual(true)}
            className="text-xs font-bold uppercase text-slate-400 flex items-center gap-2 hover:text-slate-600"
          >
            <Wrench size={14} /> Sin conexión: registrar manualmente
          </button>
          {pendientes > 0 && (
            <button
              onClick={sincronizar}
              disabled={sincronizando}
              className="text-xs font-bold uppercase text-amber-600 flex items-center gap-2 hover:text-amber-700"
              title="Reintentar sincronización ahora"
            >
              {sincronizando ? <RefreshCw size={14} className="animate-spin" /> : <CloudOff size={14} />}
              {pendientes} pendiente{pendientes === 1 ? '' : 's'}
            </button>
          )}
        </div>

        {resultado && (
          <div
            className={`w-full max-w-sm p-6 rounded-[2rem] border-2 bg-white shadow-lg ${
              resultado.type === 'success' ? 'border-emerald-100' : 'border-rose-100'
            }`}
          >
            {resultado.type === 'success' ? (
              <div className="flex items-start gap-3">
                {resultado.offline ? (
                  <CloudUpload className="text-amber-500 shrink-0" size={24} />
                ) : (
                  <CheckCircle className="text-emerald-500 shrink-0" size={24} />
                )}
                <div>
                  <p className="font-black text-slate-900">{resultado.data.mensaje}</p>
                  <p className="text-xs font-bold uppercase tracking-widest mt-1 text-slate-400">
                    {resultado.data.tipo} · {resultado.data.estado || 'sin horario asignado'}
                  </p>
                </div>
              </div>
            ) : (
              <div className="flex items-start gap-3">
                <AlertCircle className="text-rose-500 shrink-0" size={24} />
                <p className="font-bold text-rose-600">{resultado.msg}</p>
              </div>
            )}
          </div>
        )}
      </div>

      <div className="bg-white p-6 rounded-[2rem] border border-slate-200 shadow-sm">
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center gap-3">
            <div className="bg-teal-100 p-2.5 rounded-xl text-teal-700"><ScanEye size={20} /></div>
            <h3 className="text-lg font-black uppercase text-slate-900">Asistencia de hoy</h3>
          </div>
          <button onClick={cargarHoy} className="text-slate-400 hover:text-teal-700">
            <RefreshCw size={18} />
          </button>
        </div>
        <div className="space-y-2 max-h-[28rem] overflow-y-auto pr-1">
          {hoy.length === 0 && (
            <p className="text-sm text-slate-400 font-bold text-center py-8">Sin empleados activos.</p>
          )}
          {hoy.map((item) => (
            <div key={item.empleado_id} className="bg-slate-50 border border-slate-100 p-3 rounded-xl flex items-center justify-between">
              <div>
                <p className="font-bold text-slate-900 text-sm">{item.nombre}</p>
                <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">{item.puesto || 'Sin puesto'}</p>
              </div>
              <div className="text-right">
                <span
                  className={`text-[10px] font-black px-2 py-1 rounded-lg uppercase ${
                    item.estatus === 'entrada' ? 'bg-emerald-100 text-emerald-700' :
                    item.estatus === 'salida' ? 'bg-slate-200 text-slate-600' :
                    'bg-amber-100 text-amber-700'
                  }`}
                >
                  {item.estatus}
                </span>
                {item.hora && <p className="text-[10px] text-slate-400 mt-1">{item.hora}</p>}
              </div>
            </div>
          ))}
        </div>
      </div>

      {mostrarManual && (
        <RegistroManualModal
          onClose={() => setMostrarManual(false)}
          onRegistrado={(offline) => {
            setMostrarManual(false);
            setPendientes(contarPendientes());
            if (!offline) cargarHoy();
          }}
        />
      )}
    </div>
  );
}

function RegistroManualModal({ onClose, onRegistrado }) {
  const [empleados, setEmpleados] = useState([]);
  const [empleadoId, setEmpleadoId] = useState('');
  const [tipo, setTipo] = useState('entrada');
  const [enviando, setEnviando] = useState(false);

  useEffect(() => {
    api.get('/empleados').then((res) => setEmpleados(res.data || [])).catch(() => {});
  }, []);

  const registrar = async (e) => {
    e.preventDefault();
    if (!empleadoId) return;
    setEnviando(true);
    try {
      await api.post('/asistencia/marcar-manual', { empleado_id: Number(empleadoId), tipo });
      onRegistrado(false);
    } catch (err) {
      if (!err.response) {
        // Sin respuesta del servidor: no hay red. Se guarda localmente en
        // vez de perder el marcaje (SPEC.md sección 8).
        const empleado = empleados.find((e) => e.id === Number(empleadoId));
        encolarMarcaje({ empleadoId: Number(empleadoId), nombre: empleado?.nombre_completo, tipo });
        alert('Sin conexión: el marcaje se guardó en este dispositivo y se sincronizará solo cuando vuelva el internet.');
        onRegistrado(true);
      } else {
        alert(err.response?.data?.error || 'No se pudo registrar');
      }
    } finally {
      setEnviando(false);
    }
  };

  return (
    <div className="fixed inset-0 z-[150] flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-md">
      <div className="bg-white w-full max-w-sm rounded-[2rem] shadow-2xl p-6">
        <h3 className="text-lg font-black uppercase text-slate-900 mb-1">Registro manual</h3>
        <p className="text-xs text-slate-400 mb-6">
          Úsalo si la cámara o Rekognition fallan. Si tampoco hay internet, se guarda en este dispositivo y se sincroniza solo.
        </p>
        <form onSubmit={registrar} className="space-y-4">
          <select
            value={empleadoId}
            onChange={(e) => setEmpleadoId(e.target.value)}
            className="w-full bg-slate-50 border border-slate-200 p-3 rounded-xl outline-none"
            required
          >
            <option value="">Selecciona un empleado</option>
            {empleados.map((emp) => (
              <option key={emp.id} value={emp.id}>{emp.nombre_completo}</option>
            ))}
          </select>
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => setTipo('entrada')}
              className={`flex-1 py-3 rounded-xl text-xs font-black uppercase ${tipo === 'entrada' ? 'bg-teal-700 text-white' : 'bg-slate-100 text-slate-400'}`}
            >
              Entrada
            </button>
            <button
              type="button"
              onClick={() => setTipo('salida')}
              className={`flex-1 py-3 rounded-xl text-xs font-black uppercase ${tipo === 'salida' ? 'bg-teal-700 text-white' : 'bg-slate-100 text-slate-400'}`}
            >
              Salida
            </button>
          </div>
          <div className="flex gap-2 pt-2">
            <button type="button" onClick={onClose} className="flex-1 py-3 text-slate-500 font-bold uppercase text-xs">
              Cancelar
            </button>
            <button
              type="submit"
              disabled={enviando}
              className="flex-1 bg-slate-900 py-3 rounded-xl font-black text-white uppercase text-xs disabled:opacity-50"
            >
              {enviando ? 'Guardando...' : 'Registrar'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
