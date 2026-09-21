import React, { useEffect, useState } from 'react';
import { TrendingUp, Download } from 'lucide-react';
import api from '../axiosConfig';

const hoyISO = () => new Date().toISOString().slice(0, 10);
const inicioDeMesISO = () => {
  const d = new Date();
  return new Date(d.getFullYear(), d.getMonth(), 1).toISOString().slice(0, 10);
};

export default function Reportes() {
  const [inicio, setInicio] = useState(inicioDeMesISO());
  const [fin, setFin] = useState(hoyISO());
  const [empleados, setEmpleados] = useState([]);
  const [empleadoId, setEmpleadoId] = useState('');
  const [reporte, setReporte] = useState(null);
  const [cargando, setCargando] = useState(false);

  useEffect(() => {
    api.get('/empleados').then((res) => setEmpleados(res.data || [])).catch(() => {});
  }, []);

  const consultar = async (e) => {
    e?.preventDefault();
    setCargando(true);
    try {
      const params = { inicio, fin };
      if (empleadoId) params.empleado_id = empleadoId;
      const res = await api.get('/reportes/asistencia', { params });
      setReporte(res.data);
    } catch (err) {
      alert(err.response?.data?.error || 'No se pudo generar el reporte');
    } finally {
      setCargando(false);
    }
  };

  useEffect(() => {
    consultar();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const exportarCSV = () => {
    if (!reporte) return;
    const filas = [['empleado', 'dias_trabajados', 'retardos', 'faltas', 'horas_trabajadas', 'horas_extra', '% puntualidad']];
    reporte.empleados.forEach((e) => {
      filas.push([
        e.nombre,
        e.resumen.dias_trabajados,
        e.resumen.retardos,
        e.resumen.faltas,
        e.resumen.horas_trabajadas,
        e.resumen.horas_extra,
        e.resumen.porcentaje_puntualidad,
      ]);
    });
    const csv = filas.map((fila) => fila.join(',')).join('\n');
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `reporte-asistencia_${reporte.inicio}_${reporte.fin}.csv`;
    link.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="space-y-6">
      <form onSubmit={consultar} className="bg-white p-6 rounded-[2rem] border border-slate-200 shadow-sm flex flex-wrap items-end gap-4">
        <label className="text-xs font-bold text-slate-500 uppercase">
          Desde
          <input type="date" value={inicio} onChange={(e) => setInicio(e.target.value)} className="block mt-1 bg-slate-50 border border-slate-200 p-3 rounded-xl outline-none" />
        </label>
        <label className="text-xs font-bold text-slate-500 uppercase">
          Hasta
          <input type="date" value={fin} onChange={(e) => setFin(e.target.value)} className="block mt-1 bg-slate-50 border border-slate-200 p-3 rounded-xl outline-none" />
        </label>
        <label className="text-xs font-bold text-slate-500 uppercase flex-1 min-w-[10rem]">
          Empleado
          <select value={empleadoId} onChange={(e) => setEmpleadoId(e.target.value)} className="block mt-1 w-full bg-slate-50 border border-slate-200 p-3 rounded-xl outline-none">
            <option value="">Todos</option>
            {empleados.map((emp) => (
              <option key={emp.id} value={emp.id}>{emp.nombre_completo}</option>
            ))}
          </select>
        </label>
        <button type="submit" disabled={cargando} className="py-3 px-6 bg-teal-700 text-white rounded-xl font-black uppercase text-xs shadow-md disabled:opacity-50">
          {cargando ? 'Consultando...' : 'Consultar'}
        </button>
        {reporte && (
          <button type="button" onClick={exportarCSV} className="py-3 px-4 bg-slate-900 text-white rounded-xl font-black uppercase text-xs shadow-md flex items-center gap-2">
            <Download size={14} /> CSV
          </button>
        )}
      </form>

      <div className="bg-white p-6 rounded-[2rem] border border-slate-200 shadow-sm">
        <div className="flex items-center gap-3 mb-6">
          <div className="bg-teal-100 p-2.5 rounded-xl text-teal-700"><TrendingUp size={20} /></div>
          <h3 className="text-lg font-black uppercase text-slate-900">
            Resumen {reporte ? `(${reporte.inicio} a ${reporte.fin})` : ''}
          </h3>
        </div>

        {!reporte || reporte.empleados.length === 0 ? (
          <p className="text-sm text-slate-400 font-bold text-center py-8">Sin datos para el rango seleccionado.</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-[10px] font-black uppercase text-slate-400 border-b border-slate-100">
                  <th className="py-2 pr-4">Empleado</th>
                  <th className="py-2 pr-4">Días trabajados</th>
                  <th className="py-2 pr-4">Retardos</th>
                  <th className="py-2 pr-4">Faltas</th>
                  <th className="py-2 pr-4">Horas trabajadas</th>
                  <th className="py-2 pr-4">Horas extra</th>
                  <th className="py-2 pr-4">% Puntualidad</th>
                </tr>
              </thead>
              <tbody>
                {reporte.empleados.map((e) => (
                  <tr key={e.empleado_id} className="border-b border-slate-50">
                    <td className="py-3 pr-4 font-bold text-slate-900">{e.nombre}</td>
                    <td className="py-3 pr-4">{e.resumen.dias_trabajados}</td>
                    <td className="py-3 pr-4">{e.resumen.retardos}</td>
                    <td className="py-3 pr-4">{e.resumen.faltas}</td>
                    <td className="py-3 pr-4">{e.resumen.horas_trabajadas}</td>
                    <td className="py-3 pr-4">{e.resumen.horas_extra}</td>
                    <td className="py-3 pr-4">{e.resumen.porcentaje_puntualidad}%</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
