import React, { useState } from 'react';
import { ShieldCheck, ScanEye, Users, Clock, TrendingUp, UserCog, LogOut } from 'lucide-react';
import Login from './pages/Login';
import Kiosco from './pages/Kiosco';
import PanelEmpleados from './pages/PanelEmpleados';
import PanelTurnos from './pages/PanelTurnos';
import Reportes from './pages/Reportes';
import PanelUsuarios from './pages/PanelUsuarios';

// El perfil "kiosco" (la tablet del mostrador) solo puede marcar asistencia;
// todo lo demás es exclusivo de "admin" — mismo límite que ya aplica el
// backend, aquí solo evita mostrar pestañas a las que igual no se puede
// entrar.
const TABS = [
  { id: 'kiosco', label: 'Kiosco', icon: ScanEye, component: Kiosco, roles: ['admin', 'kiosco'] },
  { id: 'empleados', label: 'Empleados', icon: Users, component: PanelEmpleados, roles: ['admin'] },
  { id: 'turnos', label: 'Turnos', icon: Clock, component: PanelTurnos, roles: ['admin'] },
  { id: 'reportes', label: 'Reportes', icon: TrendingUp, component: Reportes, roles: ['admin'] },
  { id: 'usuarios', label: 'Usuarios', icon: UserCog, component: PanelUsuarios, roles: ['admin'] },
];

export default function App() {
  const [isLoggedIn, setIsLoggedIn] = useState(() => !!localStorage.getItem('token'));
  const [rol, setRol] = useState(() => localStorage.getItem('rol') || '');
  const [negocioNombre, setNegocioNombre] = useState(() => localStorage.getItem('negocio_nombre') || '');
  const [tab, setTab] = useState('kiosco');

  const cerrarSesion = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('rol');
    localStorage.removeItem('negocio_nombre');
    setIsLoggedIn(false);
  };

  const handleLogin = (data) => {
    if (data?.negocio_nombre) {
      localStorage.setItem('negocio_nombre', data.negocio_nombre);
      setNegocioNombre(data.negocio_nombre);
    }
    if (data?.rol) {
      localStorage.setItem('rol', data.rol);
      setRol(data.rol);
    }
    setIsLoggedIn(true);
    setTab('kiosco');
  };

  if (!isLoggedIn) {
    return <Login onLogin={handleLogin} />;
  }

  const tabsVisibles = TABS.filter((t) => t.roles.includes(rol));
  const ActiveComponent = tabsVisibles.find((t) => t.id === tab)?.component ?? Kiosco;

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900 p-4 sm:p-6 lg:p-8">
      <header className="flex flex-col items-center mb-8 border-b border-slate-200 pb-6 gap-6 w-full">
        <div className="flex items-center gap-3">
          <div className="bg-teal-700 p-2 rounded-xl shadow-md"><ShieldCheck size={20} className="text-white" /></div>
          <div className="text-center sm:text-left">
            <h1 className="text-xl font-black uppercase leading-none text-slate-900">StaffAttendance</h1>
            {negocioNombre && <p className="text-[9px] text-teal-700 font-bold uppercase tracking-widest">{negocioNombre}</p>}
          </div>
        </div>

        <nav className="w-full max-w-full overflow-x-auto">
          <div className="flex bg-white p-1.5 rounded-2xl border border-slate-200 shadow-sm min-w-max mx-auto">
            {tabsVisibles.length > 1 && tabsVisibles.map((item) => {
              const TabIcon = item.icon;
              return (
                <button
                  key={item.id}
                  onClick={() => setTab(item.id)}
                  className={`px-4 py-2.5 rounded-xl flex items-center gap-2 font-bold transition-all ${
                    tab === item.id ? 'bg-teal-700 text-white shadow-md' : 'text-slate-500 hover:bg-slate-50'
                  }`}
                >
                  <TabIcon size={18} /> {item.label}
                </button>
              );
            })}
            <button onClick={cerrarSesion} className="px-3 py-2 text-rose-500 hover:bg-rose-50 rounded-xl ml-2 border-l border-slate-100">
              <LogOut size={18} />
            </button>
          </div>
        </nav>
      </header>

      <main className="max-w-5xl mx-auto">
        <ActiveComponent />
      </main>
    </div>
  );
}
