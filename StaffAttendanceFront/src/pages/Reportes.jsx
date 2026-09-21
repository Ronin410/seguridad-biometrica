import React from 'react';
import { TrendingUp } from 'lucide-react';

// Reportes y exportación (SPEC.md sección 6.4) dependen del historial de
// registros_asistencia que se llenará una vez implementado el marcaje.
export default function Reportes() {
  return (
    <div className="flex flex-col items-center justify-center text-center p-16 border-2 border-dashed border-slate-300 rounded-[3rem] text-slate-400 bg-white">
      <TrendingUp size={48} className="mb-4 opacity-20" />
      <p className="font-bold uppercase text-sm tracking-widest text-slate-500">Reportes de asistencia</p>
      <p className="text-xs mt-2 max-w-sm">
        Pendiente de implementación: reportes por empleado/periodo y exportación a
        Excel/PDF/CSV (SPEC.md sección 6.4).
      </p>
    </div>
  );
}
