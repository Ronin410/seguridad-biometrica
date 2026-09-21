package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	appdb "staffattendance/internal/db"
	"staffattendance/internal/rekognition"
)

// setupRouter conecta a un Postgres real (TEST_DATABASE_URL o el default de
// desarrollo local), corre las migraciones, limpia las tablas y arma un
// router con las rutas que no dependen de una llamada real a AWS
// Rekognition (todo excepto /asistencia/marcar y /empleados/:id/enrolar).
func setupRouter(t *testing.T) (*gin.Engine, *sql.DB) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/staffattendance_test?sslmode=disable"
	}

	conn, err := appdb.Connect(dsn)
	if err != nil {
		t.Skipf("saltando pruebas de integración: no hay Postgres disponible (%v)", err)
	}

	if err := appdb.RunMigrations(conn); err != nil {
		t.Fatalf("migraciones fallaron: %v", err)
	}

	for _, table := range []string{"registros_asistencia", "empleados", "turnos"} {
		if _, err := conn.Exec("DELETE FROM " + table); err != nil {
			t.Fatalf("no se pudo limpiar %s: %v", table, err)
		}
	}

	rekClient, err := rekognition.NewClient(context.Background(), "us-east-1", "test-collection")
	if err != nil {
		t.Fatalf("no se pudo crear el cliente de rekognition: %v", err)
	}

	loc, _ := time.LoadLocation("America/Mazatlan")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewTurnosHandler(conn).Register(router)
	NewEmpleadosHandler(conn, rekClient).Register(router)
	NewAsistenciaHandler(conn, rekClient, loc, 95).Register(router)

	return router, conn
}

func doJSON(t *testing.T, router *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("no se pudo serializar body: %v", err)
		}
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder, target interface{}) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), target); err != nil {
		t.Fatalf("no se pudo decodificar respuesta (%s): %v", rec.Body.String(), err)
	}
}

func TestFlujoCompletoAsistencia(t *testing.T) {
	router, _ := setupRouter(t)

	// 1. Crear un turno cuya hora_entrada quedó hace 2 horas, con 5 min de
	// tolerancia: cualquier marcaje "ahora" debe salir como retardo.
	loc, _ := time.LoadLocation("America/Mazatlan")
	ahora := time.Now().In(loc)
	horaEntradaPasada := ahora.Add(-2 * time.Hour).Format("15:04:05")
	horaSalidaFutura := ahora.Add(2 * time.Hour).Format("15:04:05")

	turnoRec := doJSON(t, router, http.MethodPost, "/turnos", map[string]interface{}{
		"nombre":                     "Prueba",
		"hora_entrada":               horaEntradaPasada,
		"hora_salida":                horaSalidaFutura,
		"tolerancia_retardo_minutos": 5,
		"dias_aplicables":            []int{0, 1, 2, 3, 4, 5, 6},
	})
	if turnoRec.Code != http.StatusCreated {
		t.Fatalf("crear turno: esperaba 201, obtuve %d (%s)", turnoRec.Code, turnoRec.Body.String())
	}
	var turno struct {
		ID int `json:"id"`
	}
	decode(t, turnoRec, &turno)

	// 2. Crear un empleado con ese turno.
	empRec := doJSON(t, router, http.MethodPost, "/empleados", map[string]interface{}{
		"nombre_completo": "Empleado de Prueba",
		"puesto":          "Cajero",
		"turno_id":        turno.ID,
	})
	if empRec.Code != http.StatusCreated {
		t.Fatalf("crear empleado: esperaba 201, obtuve %d (%s)", empRec.Code, empRec.Body.String())
	}
	var empleado struct {
		ID int `json:"id"`
	}
	decode(t, empRec, &empleado)

	// 3. Marcar entrada manualmente: como la hora_entrada + tolerancia ya
	// pasó, debe quedar como retardo.
	entradaRec := doJSON(t, router, http.MethodPost, "/asistencia/marcar-manual", map[string]interface{}{
		"empleado_id": empleado.ID,
		"tipo":        "entrada",
	})
	if entradaRec.Code != http.StatusOK {
		t.Fatalf("marcar entrada: esperaba 200, obtuve %d (%s)", entradaRec.Code, entradaRec.Body.String())
	}
	var entrada struct {
		Tipo   string `json:"tipo"`
		Estado string `json:"estado"`
	}
	decode(t, entradaRec, &entrada)
	if entrada.Tipo != "entrada" {
		t.Errorf("tipo = %q, quería 'entrada'", entrada.Tipo)
	}
	if entrada.Estado != "retardo" {
		t.Errorf("estado = %q, quería 'retardo' (turno empezó hace 2h con 5 min de tolerancia)", entrada.Estado)
	}

	// 4. Marcar de nuevo inmediatamente: debe ser rechazado como duplicado.
	dupRec := doJSON(t, router, http.MethodPost, "/asistencia/marcar-manual", map[string]interface{}{
		"empleado_id": empleado.ID,
		"tipo":        "salida",
	})
	if dupRec.Code != http.StatusConflict {
		t.Fatalf("marcaje duplicado: esperaba 409, obtuve %d (%s)", dupRec.Code, dupRec.Body.String())
	}

	// 5. GET /asistencia/hoy debe reflejar al empleado con estatus "entrada".
	hoyRec := doJSON(t, router, http.MethodGet, "/asistencia/hoy", nil)
	if hoyRec.Code != http.StatusOK {
		t.Fatalf("asistencia/hoy: esperaba 200, obtuve %d (%s)", hoyRec.Code, hoyRec.Body.String())
	}
	var hoy []struct {
		EmpleadoID int    `json:"empleado_id"`
		Estatus    string `json:"estatus"`
	}
	decode(t, hoyRec, &hoy)
	if len(hoy) != 1 || hoy[0].EmpleadoID != empleado.ID || hoy[0].Estatus != "entrada" {
		t.Fatalf("asistencia/hoy inesperado: %+v", hoy)
	}

	// 6. El reporte del día debe contar 1 día trabajado y 1 retardo.
	fechaHoy := ahora.Format("2006-01-02")
	reporteRec := doJSON(t, router, http.MethodGet, "/reportes/asistencia?inicio="+fechaHoy+"&fin="+fechaHoy, nil)
	if reporteRec.Code != http.StatusOK {
		t.Fatalf("reporte: esperaba 200, obtuve %d (%s)", reporteRec.Code, reporteRec.Body.String())
	}
	var reporte struct {
		Empleados []struct {
			EmpleadoID int `json:"empleado_id"`
			Resumen    struct {
				DiasTrabajados int `json:"dias_trabajados"`
				Retardos       int `json:"retardos"`
				Faltas         int `json:"faltas"`
			} `json:"resumen"`
		} `json:"empleados"`
	}
	decode(t, reporteRec, &reporte)
	if len(reporte.Empleados) != 1 {
		t.Fatalf("reporte: esperaba 1 empleado, obtuve %d", len(reporte.Empleados))
	}
	resumen := reporte.Empleados[0].Resumen
	if resumen.DiasTrabajados != 1 || resumen.Retardos != 1 || resumen.Faltas != 0 {
		t.Fatalf("resumen inesperado: %+v", resumen)
	}
}

func TestListarTurnosDevuelveDiasAplicablesYHoraPlano(t *testing.T) {
	router, _ := setupRouter(t)

	doJSON(t, router, http.MethodPost, "/turnos", map[string]interface{}{
		"nombre":                     "Vespertino",
		"hora_entrada":               "14:00:00",
		"hora_salida":                "22:00:00",
		"tolerancia_retardo_minutos": 15,
		"dias_aplicables":            []int{1, 3, 5},
	})

	rec := doJSON(t, router, http.MethodGet, "/turnos", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("listar turnos: esperaba 200, obtuve %d (%s)", rec.Code, rec.Body.String())
	}

	var turnos []struct {
		Nombre         string `json:"nombre"`
		HoraEntrada    string `json:"hora_entrada"`
		DiasAplicables []int  `json:"dias_aplicables"`
	}
	decode(t, rec, &turnos)
	if len(turnos) != 1 {
		t.Fatalf("esperaba 1 turno, obtuve %d", len(turnos))
	}
	if len(turnos[0].DiasAplicables) != 3 {
		t.Fatalf("dias_aplicables vino vacío o incompleto: %+v (regresión del bug de escaneo de arrays de Postgres)", turnos[0])
	}
	// lib/pq decodifica columnas TIME como time.Time con fecha 0000-01-01;
	// si se escanea directo a *string, Go lo reformatea a RFC3339
	// ("0000-01-01T14:00:00Z") en vez de "14:00:00". Este assert es la
	// regresión de ese bug.
	if turnos[0].HoraEntrada != "14:00:00" {
		t.Fatalf("hora_entrada = %q, quería \"14:00:00\" (regresión del bug de escaneo de TIME de Postgres)", turnos[0].HoraEntrada)
	}
}

// TestClasificacionPuntualVsRetardo verifica ambas ramas de clasificarEntrada
// usando el endpoint real (no la función en aislado), para que cubra también
// el bug de decodificación de columnas TIME al leer el turno desde la BD.
func TestClasificacionPuntualVsRetardo(t *testing.T) {
	router, _ := setupRouter(t)
	loc, _ := time.LoadLocation("America/Mazatlan")
	ahora := time.Now().In(loc)

	crearTurnoYEmpleado := func(nombre, horaEntrada, horaSalida string, tolerancia int) int {
		turnoRec := doJSON(t, router, http.MethodPost, "/turnos", map[string]interface{}{
			"nombre":                     nombre,
			"hora_entrada":               horaEntrada,
			"hora_salida":                horaSalida,
			"tolerancia_retardo_minutos": tolerancia,
			"dias_aplicables":            []int{0, 1, 2, 3, 4, 5, 6},
		})
		var turno struct {
			ID int `json:"id"`
		}
		decode(t, turnoRec, &turno)

		empRec := doJSON(t, router, http.MethodPost, "/empleados", map[string]interface{}{
			"nombre_completo": nombre,
			"turno_id":        turno.ID,
		})
		var empleado struct {
			ID int `json:"id"`
		}
		decode(t, empRec, &empleado)
		return empleado.ID
	}

	// Turno que empieza dentro de 3 horas: marcar "ahora" debe ser puntual
	// (llega mucho antes del límite de tolerancia).
	empleadoPuntual := crearTurnoYEmpleado("Puntual", ahora.Add(3*time.Hour).Format("15:04:05"), ahora.Add(11*time.Hour).Format("15:04:05"), 10)
	// Turno que empezó hace 3 horas: marcar "ahora" debe ser retardo.
	empleadoRetardo := crearTurnoYEmpleado("Retardo", ahora.Add(-3*time.Hour).Format("15:04:05"), ahora.Add(5*time.Hour).Format("15:04:05"), 10)

	rec1 := doJSON(t, router, http.MethodPost, "/asistencia/marcar-manual", map[string]interface{}{"empleado_id": empleadoPuntual, "tipo": "entrada"})
	var r1 struct {
		Estado string `json:"estado"`
	}
	decode(t, rec1, &r1)
	if r1.Estado != "puntual" {
		t.Errorf("empleado con turno futuro: estado = %q, quería 'puntual'", r1.Estado)
	}

	rec2 := doJSON(t, router, http.MethodPost, "/asistencia/marcar-manual", map[string]interface{}{"empleado_id": empleadoRetardo, "tipo": "entrada"})
	var r2 struct {
		Estado string `json:"estado"`
	}
	decode(t, rec2, &r2)
	if r2.Estado != "retardo" {
		t.Errorf("empleado con turno pasado: estado = %q, quería 'retardo'", r2.Estado)
	}
}

func TestReporteMarcaFaltaSinRegistro(t *testing.T) {
	router, _ := setupRouter(t)
	loc, _ := time.LoadLocation("America/Mazatlan")
	ayer := time.Now().In(loc).AddDate(0, 0, -1)

	turnoRec := doJSON(t, router, http.MethodPost, "/turnos", map[string]interface{}{
		"nombre":                     "TodosLosDias",
		"hora_entrada":               "08:00:00",
		"hora_salida":                "17:00:00",
		"tolerancia_retardo_minutos": 10,
		"dias_aplicables":            []int{0, 1, 2, 3, 4, 5, 6},
	})
	var turno struct {
		ID int `json:"id"`
	}
	decode(t, turnoRec, &turno)

	empRec := doJSON(t, router, http.MethodPost, "/empleados", map[string]interface{}{
		"nombre_completo": "Empleado Sin Registros",
		"turno_id":        turno.ID,
	})
	var empleado struct {
		ID int `json:"id"`
	}
	decode(t, empRec, &empleado)

	inicio := ayer.Format("2006-01-02")
	reporteRec := doJSON(t, router, http.MethodGet, "/reportes/asistencia?inicio="+inicio+"&fin="+inicio+"&empleado_id="+strconv.Itoa(empleado.ID), nil)
	if reporteRec.Code != http.StatusOK {
		t.Fatalf("reporte: esperaba 200, obtuve %d (%s)", reporteRec.Code, reporteRec.Body.String())
	}
	var reporte struct {
		Empleados []struct {
			Resumen struct {
				Faltas int `json:"faltas"`
			} `json:"resumen"`
		} `json:"empleados"`
	}
	decode(t, reporteRec, &reporte)
	if len(reporte.Empleados) != 1 || reporte.Empleados[0].Resumen.Faltas != 1 {
		t.Fatalf("esperaba 1 falta para el día sin registros, obtuve: %+v", reporte)
	}
}
