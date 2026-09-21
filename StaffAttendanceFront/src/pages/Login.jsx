import React, { useState } from 'react';
import { ShieldCheck } from 'lucide-react';
import api from '../axiosConfig';

export default function Login({ onLogin }) {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');

  const handleSubmit = async (e) => {
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

  return (
    <div className="min-h-screen bg-slate-100 flex items-center justify-center p-4">
      <div className="bg-white border border-slate-200 p-8 rounded-[2rem] w-full max-w-md shadow-xl text-center">
        <div className="inline-block bg-teal-700 p-4 rounded-3xl shadow-lg mb-6">
          <ShieldCheck size={40} className="text-white" />
        </div>
        <h1 className="text-2xl font-black text-slate-900 uppercase mb-1">StaffAttendance</h1>
        <p className="text-slate-500 font-bold uppercase text-[10px] tracking-widest mb-6">
          Panel administrativo
        </p>

        <form onSubmit={handleSubmit} className="space-y-4">
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
      </div>
    </div>
  );
}
