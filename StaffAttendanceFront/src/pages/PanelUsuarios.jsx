import React, { useEffect, useState } from 'react';
import { UserCog, Tablet, ShieldCheck } from 'lucide-react';
import api from '../axiosConfig';

export default function PanelUsuarios() {
  const [usuarios, setUsuarios] = useState([]);
  const [form, setForm] = useState({ username: '', password: '', rol: 'kiosco' });
  const [error, setError] = useState('');
  const [enviando, setEnviando] = useState(false);

  const cargarUsuarios = async () => {
    try {
      const res = await api.get('/usuarios');
      setUsuarios(res.data || []);
    } catch (err) {
      console.error(err);
    }
  };

  useEffect(() => {
    cargarUsuarios();
  }, []);

  const crearUsuario = async (e) => {
    e.preventDefault();
    setError('');
    setEnviando(true);
    try {
      await api.post('/usuarios/registro', form);
      setForm({ username: '', password: '', rol: 'kiosco' });
      cargarUsuarios();
    } catch (err) {
      setError(err.response?.data?.error || 'No se pudo crear el usuario');
    } finally {
      setEnviando(false);
    }
  };

  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <div className="bg-white p-6 rounded-[2rem] border border-slate-200 shadow-sm">
        <div className="flex items-center gap-3 mb-2">
          <div className="bg-teal-100 p-2.5 rounded-xl text-teal-700"><UserCog size={20} /></div>
          <h3 className="text-lg font-black uppercase text-slate-900">Nuevo usuario</h3>
        </div>
        <p className="text-xs text-slate-400 mb-6">
          Usa <b>Kiosco</b> para la cuenta de la tablet del mostrador: solo puede marcar asistencia, nunca ver empleados, turnos o reportes.
        </p>
        <form onSubmit={crearUsuario} className="space-y-4">
          <input
            type="text"
            placeholder="Usuario"
            value={form.username}
            onChange={(e) => setForm({ ...form, username: e.target.value })}
            className="w-full bg-slate-50 border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-teal-600"
            required
          />
          <input
            type="password"
            placeholder="Contraseña"
            value={form.password}
            onChange={(e) => setForm({ ...form, password: e.target.value })}
            className="w-full bg-slate-50 border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-teal-600"
            required
          />
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => setForm({ ...form, rol: 'kiosco' })}
              className={`flex-1 py-3 rounded-xl text-xs font-black uppercase flex items-center justify-center gap-2 ${form.rol === 'kiosco' ? 'bg-teal-700 text-white' : 'bg-slate-100 text-slate-400'}`}
            >
              <Tablet size={14} /> Kiosco
            </button>
            <button
              type="button"
              onClick={() => setForm({ ...form, rol: 'admin' })}
              className={`flex-1 py-3 rounded-xl text-xs font-black uppercase flex items-center justify-center gap-2 ${form.rol === 'admin' ? 'bg-teal-700 text-white' : 'bg-slate-100 text-slate-400'}`}
            >
              <ShieldCheck size={14} /> Admin
            </button>
          </div>
          {error && <p className="text-rose-600 text-sm font-bold">{error}</p>}
          <button
            type="submit"
            disabled={enviando}
            className="w-full py-3 bg-slate-900 text-white rounded-xl font-black uppercase text-sm shadow-md disabled:opacity-50"
          >
            {enviando ? 'Guardando...' : 'Crear usuario'}
          </button>
        </form>
      </div>

      <div className="bg-white p-6 rounded-[2rem] border border-slate-200 shadow-sm">
        <div className="flex items-center gap-3 mb-6">
          <div className="bg-teal-100 p-2.5 rounded-xl text-teal-700"><UserCog size={20} /></div>
          <h3 className="text-lg font-black uppercase text-slate-900">Usuarios del negocio</h3>
        </div>
        <div className="space-y-3 max-h-[28rem] overflow-y-auto pr-1">
          {usuarios.length === 0 && (
            <p className="text-sm text-slate-400 font-bold text-center py-8">Aún no hay más usuarios.</p>
          )}
          {usuarios.map((u) => (
            <div key={u.id} className="bg-slate-50 border border-slate-100 p-4 rounded-xl flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className={`p-2 rounded-lg ${u.rol === 'admin' ? 'bg-violet-100 text-violet-600' : 'bg-teal-100 text-teal-700'}`}>
                  {u.rol === 'admin' ? <ShieldCheck size={16} /> : <Tablet size={16} />}
                </div>
                <p className="font-bold text-slate-900 text-sm">{u.username}</p>
              </div>
              <span className={`text-[10px] font-black px-2 py-1 rounded-lg uppercase ${u.rol === 'admin' ? 'bg-violet-100 text-violet-600' : 'bg-teal-100 text-teal-700'}`}>
                {u.rol}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
