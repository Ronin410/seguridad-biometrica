import React, { useState } from 'react';
import { ShieldCheck } from 'lucide-react';
import api from '../axiosConfig';

export default function Login({ onLogin }) {
  const [modo, setModo] = useState('login'); // 'login' | 'crear'
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [negocioNombre, setNegocioNombre] = useState('');
  const [error, setError] = useState('');
  const [enviando, setEnviando] = useState(false);

  const handleLogin = async (e) => {
    e.preventDefault();
    setError('');
    try {
      const res = await api.post('/login', { username, password });
      localStorage.setItem('token', res.data.token);
      onLogin(res.data);
    } catch (err) {
      setError(err.response?.data?.error || 'No se pudo iniciar sesión');
    }
  };

  const handleCrearNegocio = async (e) => {
    e.preventDefault();
    setError('');
    setEnviando(true);
    try {
      const res = await api.post('/negocios', {
        negocio_nombre: negocioNombre,
        username,
        password,
      });
      localStorage.setItem('token', res.data.token);
      onLogin(res.data);
    } catch (err) {
      setError(err.response?.data?.error || 'No se pudo crear el negocio');
    } finally {
      setEnviando(false);
    }
  };

  return (
    <div className="min-h-screen bg-slate-100 flex items-center justify-center p-4">
      <div className="bg-white border border-slate-200 p-8 rounded-[2rem] w-full max-w-md shadow-xl text-center">
        <div className="inline-block bg-teal-700 p-4 rounded-3xl shadow-lg mb-6">
          <ShieldCheck size={40} className="text-white" />
        </div>
        <h1 className="text-2xl font-black text-slate-900 uppercase mb-1">StaffAttendance</h1>
        <p className="text-slate-500 font-bold uppercase text-[10px] tracking-widest mb-6">
          {modo === 'login' ? 'Panel administrativo' : 'Crear cuenta de negocio'}
        </p>

        {modo === 'login' ? (
          <form onSubmit={handleLogin} className="space-y-4">
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="w-full bg-slate-50 border border-slate-200 p-4 rounded-2xl outline-none focus:ring-2 focus:ring-teal-600"
              placeholder="Usuario"
            />
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full bg-slate-50 border border-slate-200 p-4 rounded-2xl outline-none focus:ring-2 focus:ring-teal-600"
              placeholder="••••••••"
            />
            {error && <p className="text-rose-600 text-sm font-bold">{error}</p>}
            <button
              type="submit"
              className="w-full bg-teal-700 hover:bg-teal-800 text-white font-black py-4 rounded-2xl uppercase tracking-tight shadow-lg transition-all active:scale-95"
            >
              Entrar
            </button>
          </form>
        ) : (
          <form onSubmit={handleCrearNegocio} className="space-y-4">
            <input
              type="text"
              value={negocioNombre}
              onChange={(e) => setNegocioNombre(e.target.value)}
              className="w-full bg-slate-50 border border-slate-200 p-4 rounded-2xl outline-none focus:ring-2 focus:ring-teal-600"
              placeholder="Nombre del negocio"
              required
            />
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="w-full bg-slate-50 border border-slate-200 p-4 rounded-2xl outline-none focus:ring-2 focus:ring-teal-600"
              placeholder="Usuario administrador"
              required
            />
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full bg-slate-50 border border-slate-200 p-4 rounded-2xl outline-none focus:ring-2 focus:ring-teal-600"
              placeholder="••••••••"
              required
            />
            {error && <p className="text-rose-600 text-sm font-bold">{error}</p>}
            <button
              type="submit"
              disabled={enviando}
              className="w-full bg-teal-700 hover:bg-teal-800 text-white font-black py-4 rounded-2xl uppercase tracking-tight shadow-lg transition-all active:scale-95 disabled:opacity-50"
            >
              {enviando ? 'Creando...' : 'Crear negocio'}
            </button>
          </form>
        )}

        <button
          onClick={() => { setModo(modo === 'login' ? 'crear' : 'login'); setError(''); }}
          className="mt-6 text-xs font-bold uppercase text-slate-400 hover:text-teal-700"
        >
          {modo === 'login' ? '¿Negocio nuevo? Crear cuenta' : '¿Ya tienes cuenta? Entrar'}
        </button>
      </div>
    </div>
  );
}
