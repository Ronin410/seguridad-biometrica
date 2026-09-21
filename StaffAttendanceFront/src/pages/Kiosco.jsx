import React from 'react';
import { ScanEye } from 'lucide-react';

// El marcaje de entrada/salida (SPEC.md sección 6.2) depende de la lógica de
// puntualidad/retardo que todavía no existe en el backend (ver
// StaffAttendanceBack/internal/handlers/asistencia.go). Esta pantalla queda
// lista para conectarse en cuanto ese endpoint esté implementado.
export default function Kiosco() {
  return (
    <div className="flex flex-col items-center justify-center text-center p-16 border-2 border-dashed border-slate-300 rounded-[3rem] text-slate-400 bg-white">
      <ScanEye size={48} className="mb-4 opacity-20" />
      <p className="font-bold uppercase text-sm tracking-widest text-slate-500">Kiosco de marcaje</p>
      <p className="text-xs mt-2 max-w-sm">
        Pendiente de implementación: reconocimiento en tiempo real y cálculo de puntualidad
        (SPEC.md sección 6.2).
      </p>
    </div>
  );
}
