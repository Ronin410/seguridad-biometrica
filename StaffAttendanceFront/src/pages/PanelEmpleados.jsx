import React, { useEffect, useRef, useState } from 'react';
import Webcam from 'react-webcam';
import { UserPlus, ScanFace, Users, Power } from 'lucide-react';
import api from '../axiosConfig';

export default function PanelEmpleados() {
  const [empleados, setEmpleados] = useState([]);
  const [turnos, setTurnos] = useState([]);
  const [form, setForm] = useState({ nombre_completo: '', puesto: '', departamento: '', turno_id: '' });
  const [empleadoParaEnrolar, setEmpleadoParaEnrolar] = useState(null);
  const webcamRef = useRef(null);
  const [enrolando, setEnrolando] = useState(false);

  const cargarEmpleados = async () => {
    try {
      const res = await api.get('/empleados');
      setEmpleados(res.data || []);
    } catch (err) {
      console.error(err);
    }
  };

  const cargarTurnos = async () => {
    try {
      const res = await api.get('/turnos');
      setTurnos(res.data || []);
    } catch (err) {
      console.error(err);
    }
  };

  useEffect(() => {
    cargarEmpleados();
    cargarTurnos();
  }, []);

  const crearEmpleado = async (e) => {
    e.preventDefault();
    try {
      await api.post('/empleados', {
        ...form,
        turno_id: form.turno_id ? Number(form.turno_id) : null,
      });
      setForm({ nombre_completo: '', puesto: '', departamento: '', turno_id: '' });
      cargarEmpleados();
    } catch (err) {
      alert(err.response?.data?.error || 'No se pudo crear el empleado');
    }
  };

  const cambiarEstado = async (empleado) => {
    const nuevoEstado = empleado.estado === 'activo' ? 'inactivo' : 'activo';
    try {
      await api.patch(`/empleados/${empleado.id}/estado`, { estado: nuevoEstado });
      cargarEmpleados();
    } catch (err) {
      alert(err.response?.data?.error || 'No se pudo actualizar el estado');
    }
  };

  const capturarYEnrolar = async () => {
    if (!webcamRef.current || !empleadoParaEnrolar) return;
    const imageSrc = webcamRef.current.getScreenshot();
    if (!imageSrc) return alert('No se pudo capturar la imagen.');

    setEnrolando(true);
    try {
      await api.post(`/empleados/${empleadoParaEnrolar.id}/enrolar`, {
        imagen: imageSrc.split(',')[1],
      });
      alert(`Rostro registrado para ${empleadoParaEnrolar.nombre_completo}`);
      setEmpleadoParaEnrolar(null);
      cargarEmpleados();
    } catch (err) {
      alert(err.response?.data?.error || 'No se pudo enrolar el rostro');
    } finally {
      setEnrolando(false);
    }
  };

  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <div className="bg-white p-6 rounded-[2rem] border border-slate-200 shadow-sm">
        <div className="flex items-center gap-3 mb-6">
          <div className="bg-teal-100 p-2.5 rounded-xl text-teal-700"><UserPlus size={20} /></div>
          <h3 className="text-lg font-black uppercase text-slate-900">Nuevo empleado</h3>
        </div>
        <form onSubmit={crearEmpleado} className="space-y-4">
          <input
            type="text"
            placeholder="Nombre completo"
            value={form.nombre_completo}
            onChange={(e) => setForm({ ...form, nombre_completo: e.target.value })}
            className="w-full bg-slate-50 border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-teal-600"
            required
          />
          <input
            type="text"
            placeholder="Puesto"
            value={form.puesto}
            onChange={(e) => setForm({ ...form, puesto: e.target.value })}
            className="w-full bg-slate-50 border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-teal-600"
          />
          <input
            type="text"
            placeholder="Departamento"
            value={form.departamento}
            onChange={(e) => setForm({ ...form, departamento: e.target.value })}
            className="w-full bg-slate-50 border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-teal-600"
          />
          <select
            value={form.turno_id}
            onChange={(e) => setForm({ ...form, turno_id: e.target.value })}
            className="w-full bg-slate-50 border border-slate-200 p-3 rounded-xl outline-none"
          >
            <option value="">Sin turno asignado</option>
            {turnos.map((t) => (
              <option key={t.id} value={t.id}>{t.nombre}</option>
            ))}
          </select>
          <button
            type="submit"
            className="w-full py-3 bg-slate-900 text-white rounded-xl font-black uppercase text-sm shadow-md"
          >
            Guardar empleado
          </button>
        </form>
      </div>

      <div className="bg-white p-6 rounded-[2rem] border border-slate-200 shadow-sm">
        <div className="flex items-center gap-3 mb-6">
          <div className="bg-teal-100 p-2.5 rounded-xl text-teal-700"><Users size={20} /></div>
          <h3 className="text-lg font-black uppercase text-slate-900">Empleados</h3>
        </div>
        <div className="space-y-3 max-h-[28rem] overflow-y-auto pr-1">
          {empleados.length === 0 && (
            <p className="text-sm text-slate-400 font-bold text-center py-8">Aún no hay empleados registrados.</p>
          )}
          {empleados.map((empleado) => (
            <div key={empleado.id} className="bg-slate-50 border border-slate-100 p-4 rounded-xl flex items-center justify-between gap-3">
              <div>
                <p className="font-black uppercase text-slate-900">{empleado.nombre_completo}</p>
                <p className="text-sm text-slate-500">{empleado.puesto || 'Sin puesto'} · {empleado.departamento || 'Sin depto.'}</p>
                <p className={`text-[10px] font-bold uppercase tracking-widest mt-1 ${empleado.rekognition_face_id ? 'text-emerald-600' : 'text-amber-600'}`}>
                  {empleado.rekognition_face_id ? 'Rostro enrolado' : 'Rostro pendiente de enrolar'}
                </p>
              </div>
              <div className="flex flex-col gap-2 items-end">
                <button
                  onClick={() => setEmpleadoParaEnrolar(empleado)}
                  className="p-2 rounded-lg bg-teal-700 text-white"
                  title="Enrolar rostro"
                >
                  <ScanFace size={16} />
                </button>
                <button
                  onClick={() => cambiarEstado(empleado)}
                  className={`p-2 rounded-lg ${empleado.estado === 'activo' ? 'bg-rose-100 text-rose-600' : 'bg-emerald-100 text-emerald-600'}`}
                  title={empleado.estado === 'activo' ? 'Dar de baja' : 'Reactivar'}
                >
                  <Power size={16} />
                </button>
              </div>
            </div>
          ))}
        </div>
      </div>

      {empleadoParaEnrolar && (
        <div className="fixed inset-0 z-[150] flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-md">
          <div className="bg-white w-full max-w-md rounded-[2rem] shadow-2xl p-6 text-center">
            <h3 className="text-lg font-black uppercase text-slate-900 mb-4">
              Enrolar a {empleadoParaEnrolar.nombre_completo}
            </h3>
            <div className="relative rounded-[2rem] overflow-hidden border-4 border-white bg-slate-200 shadow-xl aspect-[3/4] mx-auto w-full max-w-xs mb-4">
              <Webcam
                audio={false}
                ref={webcamRef}
                screenshotFormat="image/jpeg"
                videoConstraints={{ facingMode: 'user' }}
                className="absolute inset-0 w-full h-full object-cover"
                mirrored
              />
            </div>
            <div className="flex gap-2">
              <button
                onClick={() => setEmpleadoParaEnrolar(null)}
                className="flex-1 py-3 text-slate-500 font-bold uppercase text-xs"
              >
                Cancelar
              </button>
              <button
                onClick={capturarYEnrolar}
                disabled={enrolando}
                className="flex-1 bg-teal-700 py-3 rounded-xl font-black text-white uppercase text-xs disabled:opacity-50"
              >
                {enrolando ? 'Procesando...' : 'Capturar rostro'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
