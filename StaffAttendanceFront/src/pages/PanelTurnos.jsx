import React, { useEffect, useState } from 'react';
import { Clock, Plus } from 'lucide-react';
import api from '../axiosConfig';

const DIAS = ['Dom', 'Lun', 'Mar', 'Mié', 'Jue', 'Vie', 'Sáb'];

export default function PanelTurnos() {
  const [turnos, setTurnos] = useState([]);
  const [form, setForm] = useState({
    nombre: '',
    hora_entrada: '08:00',
    hora_salida: '17:00',
    tolerancia_retardo_minutos: 10,
    dias_aplicables: [1, 2, 3, 4, 5],
  });

  const cargarTurnos = async () => {
    try {
      const res = await api.get('/turnos');
      setTurnos(res.data || []);
    } catch (err) {
      console.error(err);
    }
  };

  useEffect(() => {
    cargarTurnos();
  }, []);

  const toggleDia = (dia) => {
    setForm((prev) => ({
      ...prev,
      dias_aplicables: prev.dias_aplicables.includes(dia)
        ? prev.dias_aplicables.filter((d) => d !== dia)
        : [...prev.dias_aplicables, dia].sort(),
    }));
  };

  const crearTurno = async (e) => {
    e.preventDefault();
    try {
      await api.post('/turnos', form);
      setForm({ ...form, nombre: '' });
      cargarTurnos();
    } catch (err) {
      alert(err.response?.data?.error || 'No se pudo crear el turno');
    }
  };

  return (
    <div className="grid gap-6 md:grid-cols-2">
      <div className="bg-white p-6 rounded-[2rem] border border-slate-200 shadow-sm">
        <div className="flex items-center gap-3 mb-6">
          <div className="bg-teal-100 p-2.5 rounded-xl text-teal-700"><Plus size={20} /></div>
          <h3 className="text-lg font-black uppercase text-slate-900">Nuevo turno</h3>
        </div>
        <form onSubmit={crearTurno} className="space-y-4">
          <input
            type="text"
            placeholder="Nombre (ej. Matutino)"
            value={form.nombre}
            onChange={(e) => setForm({ ...form, nombre: e.target.value })}
            className="w-full bg-slate-50 border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-teal-600"
            required
          />
          <div className="grid grid-cols-2 gap-3">
            <label className="text-xs font-bold text-slate-500 uppercase block">
              Entrada
              <input
                type="time"
                value={form.hora_entrada}
                onChange={(e) => setForm({ ...form, hora_entrada: e.target.value })}
                className="w-full mt-1 bg-slate-50 border border-slate-200 p-3 rounded-xl outline-none"
              />
            </label>
            <label className="text-xs font-bold text-slate-500 uppercase block">
              Salida
              <input
                type="time"
                value={form.hora_salida}
                onChange={(e) => setForm({ ...form, hora_salida: e.target.value })}
                className="w-full mt-1 bg-slate-50 border border-slate-200 p-3 rounded-xl outline-none"
              />
            </label>
          </div>
          <label className="text-xs font-bold text-slate-500 uppercase block">
            Tolerancia de retardo (minutos)
            <input
              type="number"
              min="0"
              value={form.tolerancia_retardo_minutos}
              onChange={(e) => setForm({ ...form, tolerancia_retardo_minutos: Number(e.target.value) })}
              className="w-full mt-1 bg-slate-50 border border-slate-200 p-3 rounded-xl outline-none"
            />
          </label>
          <div>
            <p className="text-xs font-bold text-slate-500 uppercase mb-2">Días aplicables</p>
            <div className="flex gap-2 flex-wrap">
              {DIAS.map((dia, index) => (
                <button
                  type="button"
                  key={dia}
                  onClick={() => toggleDia(index)}
                  className={`px-3 py-2 rounded-lg text-xs font-black uppercase border ${
                    form.dias_aplicables.includes(index)
                      ? 'bg-teal-700 text-white border-teal-700'
                      : 'bg-white text-slate-400 border-slate-200'
                  }`}
                >
                  {dia}
                </button>
              ))}
            </div>
          </div>
          <button
            type="submit"
            className="w-full py-3 bg-slate-900 text-white rounded-xl font-black uppercase text-sm shadow-md"
          >
            Guardar turno
          </button>
        </form>
      </div>

      <div className="bg-white p-6 rounded-[2rem] border border-slate-200 shadow-sm">
        <div className="flex items-center gap-3 mb-6">
          <div className="bg-teal-100 p-2.5 rounded-xl text-teal-700"><Clock size={20} /></div>
          <h3 className="text-lg font-black uppercase text-slate-900">Turnos configurados</h3>
        </div>
        <div className="space-y-3 max-h-[28rem] overflow-y-auto pr-1">
          {turnos.length === 0 && (
            <p className="text-sm text-slate-400 font-bold text-center py-8">Aún no hay turnos configurados.</p>
          )}
          {turnos.map((turno) => (
            <div key={turno.id} className="bg-slate-50 border border-slate-100 p-4 rounded-xl">
              <p className="font-black uppercase text-slate-900">{turno.nombre}</p>
              <p className="text-sm text-slate-500">
                {turno.hora_entrada} — {turno.hora_salida} · tolerancia {turno.tolerancia_retardo_minutos} min
              </p>
              <p className="text-[10px] font-bold text-teal-700 uppercase tracking-widest mt-1">
                {(turno.dias_aplicables || []).map((d) => DIAS[d]).join(' · ')}
              </p>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
