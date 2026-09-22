package handlers

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	appdb "staffattendance/internal/db"
	"staffattendance/internal/middleware"
	"staffattendance/internal/rekognition"
)

var testJWTSecret = []byte("test-secret")

// setupRouter conecta a un Postgres real (TEST_DATABASE_URL o el default de
// desarrollo local), corre las migraciones, limpia las tablas y arma el
// router completo (negocios, auth, empleados, turnos, asistencia) con el
// middleware de auth real. rek es el FaceRecognizer a usar — en las pruebas
// siempre un doble simulado (fakeRekognition), nunca el cliente real de AWS,
// para no depender de credenciales ni de red.
func setupRouter(t *testing.T, rek rekognition.FaceRecognizer) (*gin.Engine, *sql.DB) {
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

	for _, table := range []string{"registros_asistencia", "empleados", "turnos", "usuarios", "negocios"} {
		if _, err := conn.Exec("DELETE FROM " + table); err != nil {
			t.Fatalf("no se pudo limpiar %s: %v", table, err)
		}
	}

	loc, _ := time.LoadLocation("America/Mazatlan")

	gin.SetMode(gin.TestMode)
	router := gin.New()

	NewNegociosHandler(conn, rek, testJWTSecret).Register(router)

	authHandler := NewAuthHandler(conn, testJWTSecret)
	authHandler.RegisterPublic(router)

	protected := router.Group("/")
	protected.Use(middleware.Auth(testJWTSecret))
	authHandler.RegisterProtected(protected)
	NewTurnosHandler(conn).Register(protected)
	NewEmpleadosHandler(conn, rek).Register(protected)
	NewAsistenciaHandler(conn, rek, loc, 95).Register(protected)

	return router, conn
}

// crearNegocio da de alta un negocio nuevo (vía el endpoint real de
// onboarding) y devuelve el token de su admin, listo para usarse en el
// header Authorization del resto de las llamadas de la prueba.
func crearNegocio(t *testing.T, router *gin.Engine, nombre string) string {
	t.Helper()
	rec := doJSON(t, router, "", http.MethodPost, "/negocios", map[string]interface{}{
		"negocio_nombre": nombre,
		"username":       "admin_" + nombre,
		"password":       "clave-super-segura",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("crear negocio %q: esperaba 201, obtuve %d (%s)", nombre, rec.Code, rec.Body.String())
	}
	var resp struct {
		Token string `json:"token"`
	}
	decode(t, rec, &resp)
	if resp.Token == "" {
		t.Fatalf("crear negocio %q: no vino token en la respuesta (%s)", nombre, rec.Body.String())
	}
	return resp.Token
}

func doJSON(t *testing.T, router *gin.Engine, token, method, path string, body interface{}) *httptest.ResponseRecorder {
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
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
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
	router, _ := setupRouter(t, nuevoFakeRekognitionAutorizado())
	token := crearNegocio(t, router, "Taqueria")

	// 1. Crear un turno cuya hora_entrada quedó hace 2 horas, con 5 min de
	// tolerancia: cualquier marcaje "ahora" debe salir como retardo.
	loc, _ := time.LoadLocation("America/Mazatlan")
	ahora := time.Now().In(loc)
	horaEntradaPasada := ahora.Add(-2 * time.Hour).Format("15:04:05")
	horaSalidaFutura := ahora.Add(2 * time.Hour).Format("15:04:05")

	turnoRec := doJSON(t, router, token, http.MethodPost, "/turnos", map[string]interface{}{
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
	empRec := doJSON(t, router, token, http.MethodPost, "/empleados", map[string]interface{}{
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
	entradaRec := doJSON(t, router, token, http.MethodPost, "/asistencia/marcar-manual", map[string]interface{}{
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
	dupRec := doJSON(t, router, token, http.MethodPost, "/asistencia/marcar-manual", map[string]interface{}{
		"empleado_id": empleado.ID,
		"tipo":        "salida",
	})
	if dupRec.Code != http.StatusConflict {
		t.Fatalf("marcaje duplicado: esperaba 409, obtuve %d (%s)", dupRec.Code, dupRec.Body.String())
	}

	// 5. GET /asistencia/hoy debe reflejar al empleado con estatus "entrada".
	hoyRec := doJSON(t, router, token, http.MethodGet, "/asistencia/hoy", nil)
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
	reporteRec := doJSON(t, router, token, http.MethodGet, "/reportes/asistencia?inicio="+fechaHoy+"&fin="+fechaHoy, nil)
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
	router, _ := setupRouter(t, nuevoFakeRekognitionAutorizado())
	token := crearNegocio(t, router, "Taqueria")

	doJSON(t, router, token, http.MethodPost, "/turnos", map[string]interface{}{
		"nombre":                     "Vespertino",
		"hora_entrada":               "14:00:00",
		"hora_salida":                "22:00:00",
		"tolerancia_retardo_minutos": 15,
		"dias_aplicables":            []int{1, 3, 5},
	})

	rec := doJSON(t, router, token, http.MethodGet, "/turnos", nil)
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
	router, _ := setupRouter(t, nuevoFakeRekognitionAutorizado())
	token := crearNegocio(t, router, "Taqueria")
	loc, _ := time.LoadLocation("America/Mazatlan")
	ahora := time.Now().In(loc)

	crearTurnoYEmpleado := func(nombre, horaEntrada, horaSalida string, tolerancia int) int {
		turnoRec := doJSON(t, router, token, http.MethodPost, "/turnos", map[string]interface{}{
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

		empRec := doJSON(t, router, token, http.MethodPost, "/empleados", map[string]interface{}{
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

	rec1 := doJSON(t, router, token, http.MethodPost, "/asistencia/marcar-manual", map[string]interface{}{"empleado_id": empleadoPuntual, "tipo": "entrada"})
	var r1 struct {
		Estado string `json:"estado"`
	}
	decode(t, rec1, &r1)
	if r1.Estado != "puntual" {
		t.Errorf("empleado con turno futuro: estado = %q, quería 'puntual'", r1.Estado)
	}

	rec2 := doJSON(t, router, token, http.MethodPost, "/asistencia/marcar-manual", map[string]interface{}{"empleado_id": empleadoRetardo, "tipo": "entrada"})
	var r2 struct {
		Estado string `json:"estado"`
	}
	decode(t, rec2, &r2)
	if r2.Estado != "retardo" {
		t.Errorf("empleado con turno pasado: estado = %q, quería 'retardo'", r2.Estado)
	}
}

func TestReporteMarcaFaltaSinRegistro(t *testing.T) {
	router, _ := setupRouter(t, nuevoFakeRekognitionAutorizado())
	token := crearNegocio(t, router, "Taqueria")
	loc, _ := time.LoadLocation("America/Mazatlan")
	ayer := time.Now().In(loc).AddDate(0, 0, -1)

	turnoRec := doJSON(t, router, token, http.MethodPost, "/turnos", map[string]interface{}{
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

	empRec := doJSON(t, router, token, http.MethodPost, "/empleados", map[string]interface{}{
		"nombre_completo": "Empleado Sin Registros",
		"turno_id":        turno.ID,
	})
	var empleado struct {
		ID int `json:"id"`
	}
	decode(t, empRec, &empleado)

	inicio := ayer.Format("2006-01-02")
	reporteRec := doJSON(t, router, token, http.MethodGet, "/reportes/asistencia?inicio="+inicio+"&fin="+inicio+"&empleado_id="+strconv.Itoa(empleado.ID), nil)
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

// TestAislamientoEntreNegocios es la prueba central del multi-negocio
// (SPEC.md sección 8): un negocio nunca debe ver empleados, turnos ni
// reportes de otro, aunque ambos usen la API al mismo tiempo.
func TestAislamientoEntreNegocios(t *testing.T) {
	router, _ := setupRouter(t, nuevoFakeRekognitionAutorizado())
	tokenA := crearNegocio(t, router, "NegocioA")
	tokenB := crearNegocio(t, router, "NegocioB")

	empARec := doJSON(t, router, tokenA, http.MethodPost, "/empleados", map[string]interface{}{"nombre_completo": "Empleado A"})
	var empA struct {
		ID int `json:"id"`
	}
	decode(t, empARec, &empA)

	empBRec := doJSON(t, router, tokenB, http.MethodPost, "/empleados", map[string]interface{}{"nombre_completo": "Empleado B"})
	if empBRec.Code != http.StatusCreated {
		t.Fatalf("crear empleado B: esperaba 201, obtuve %d (%s)", empBRec.Code, empBRec.Body.String())
	}

	// El negocio A solo debe ver su propio empleado.
	listaARec := doJSON(t, router, tokenA, http.MethodGet, "/empleados", nil)
	var listaA []struct {
		NombreCompleto string `json:"nombre_completo"`
	}
	decode(t, listaARec, &listaA)
	if len(listaA) != 1 || listaA[0].NombreCompleto != "Empleado A" {
		t.Fatalf("el negocio A vio empleados que no son suyos: %+v", listaA)
	}

	// El negocio B no puede leer, editar ni marcar asistencia del empleado
	// de A usando su propio id.
	if rec := doJSON(t, router, tokenB, http.MethodPatch, fmt.Sprintf("/empleados/%d/estado", empA.ID), map[string]interface{}{"estado": "inactivo"}); rec.Code != http.StatusNotFound {
		t.Errorf("negocio B pudo desactivar un empleado de A: código %d (%s)", rec.Code, rec.Body.String())
	}
	if rec := doJSON(t, router, tokenB, http.MethodPost, "/asistencia/marcar-manual", map[string]interface{}{"empleado_id": empA.ID, "tipo": "entrada"}); rec.Code != http.StatusNotFound {
		t.Errorf("negocio B pudo marcar asistencia de un empleado de A: código %d (%s)", rec.Code, rec.Body.String())
	}

	// El reporte de B, sin filtrar por empleado, no debe traer al empleado de A.
	hoy := time.Now().Format("2006-01-02")
	reporteBRec := doJSON(t, router, tokenB, http.MethodGet, "/reportes/asistencia?inicio="+hoy+"&fin="+hoy, nil)
	var reporteB struct {
		Empleados []struct {
			Nombre string `json:"nombre"`
		} `json:"empleados"`
	}
	decode(t, reporteBRec, &reporteB)
	for _, e := range reporteB.Empleados {
		if e.Nombre == "Empleado A" {
			t.Fatalf("el reporte del negocio B incluyó al empleado de A: %+v", reporteB.Empleados)
		}
	}
}

// TestMarcarManualConCapturadoEn cubre la sincronización de la cola offline
// de la PWA (SPEC.md sección 8): el marcaje debe registrarse con la hora en
// que se capturó en la tablet, no con la hora en que llega al servidor.
func TestMarcarManualConCapturadoEn(t *testing.T) {
	router, _ := setupRouter(t, nuevoFakeRekognitionAutorizado())
	token := crearNegocio(t, router, "Taqueria")
	loc, _ := time.LoadLocation("America/Mazatlan")

	turnoRec := doJSON(t, router, token, http.MethodPost, "/turnos", map[string]interface{}{
		"nombre":                     "Matutino",
		"hora_entrada":               "08:00:00",
		"hora_salida":                "17:00:00",
		"tolerancia_retardo_minutos": 10,
		"dias_aplicables":            []int{0, 1, 2, 3, 4, 5, 6},
	})
	var turno struct {
		ID int `json:"id"`
	}
	decode(t, turnoRec, &turno)

	empRec := doJSON(t, router, token, http.MethodPost, "/empleados", map[string]interface{}{
		"nombre_completo": "Empleado Offline",
		"turno_id":        turno.ID,
	})
	var empleado struct {
		ID int `json:"id"`
	}
	decode(t, empRec, &empleado)

	// Simula que la tablet capturó el marcaje AYER a las 08:30 (20 min
	// tarde con 10 de tolerancia) mientras estaba sin internet, y recién
	// ahora ("hoy") logra sincronizar.
	ayer := time.Now().In(loc).AddDate(0, 0, -1)
	capturadoEn := time.Date(ayer.Year(), ayer.Month(), ayer.Day(), 8, 30, 0, 0, loc)

	rec := doJSON(t, router, token, http.MethodPost, "/asistencia/marcar-manual", map[string]interface{}{
		"empleado_id":  empleado.ID,
		"tipo":         "entrada",
		"capturado_en": capturadoEn.Format(time.RFC3339),
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("marcar-manual con capturado_en: esperaba 200, obtuve %d (%s)", rec.Code, rec.Body.String())
	}
	var resp struct {
		Estado string `json:"estado"`
	}
	decode(t, rec, &resp)
	if resp.Estado != "retardo" {
		t.Fatalf("estado = %q, quería 'retardo' (se capturó 20 min tarde según capturado_en, no según la hora de sincronización)", resp.Estado)
	}

	// El reporte de AYER debe traer el día trabajado; el de HOY no debe
	// traer nada, aunque la sincronización haya ocurrido hoy.
	fechaAyer := ayer.Format("2006-01-02")
	reporteAyerRec := doJSON(t, router, token, http.MethodGet, "/reportes/asistencia?inicio="+fechaAyer+"&fin="+fechaAyer+"&empleado_id="+strconv.Itoa(empleado.ID), nil)
	var reporteAyer struct {
		Empleados []struct {
			Resumen struct {
				DiasTrabajados int `json:"dias_trabajados"`
			} `json:"resumen"`
		} `json:"empleados"`
	}
	decode(t, reporteAyerRec, &reporteAyer)
	if len(reporteAyer.Empleados) != 1 || reporteAyer.Empleados[0].Resumen.DiasTrabajados != 1 {
		t.Fatalf("el registro no quedó fechado ayer (según capturado_en): %+v", reporteAyer)
	}

	fechaHoy := time.Now().In(loc).Format("2006-01-02")
	reporteHoyRec := doJSON(t, router, token, http.MethodGet, "/reportes/asistencia?inicio="+fechaHoy+"&fin="+fechaHoy+"&empleado_id="+strconv.Itoa(empleado.ID), nil)
	var reporteHoy struct {
		Empleados []struct {
			Resumen struct {
				DiasTrabajados int `json:"dias_trabajados"`
			} `json:"resumen"`
		} `json:"empleados"`
	}
	decode(t, reporteHoyRec, &reporteHoy)
	if len(reporteHoy.Empleados) == 1 && reporteHoy.Empleados[0].Resumen.DiasTrabajados != 0 {
		t.Fatalf("el registro se contó en el día de HOY en vez de en la fecha de captura: %+v", reporteHoy)
	}
}

// TestEnrolarYMarcarConAwsSimulado ejercita /empleados/:id/enrolar y
// /asistencia/marcar de punta a punta usando fakeRekognition en modo
// autorizado: no llama a AWS de verdad, pero sí prueba el flujo completo de
// reconocimiento facial (enrolar con una imagen, reconocer con la misma
// imagen, fallar con una distinta) que las demás pruebas no cubren porque
// dependían del SDK real.
func TestEnrolarYMarcarConAwsSimulado(t *testing.T) {
	router, _ := setupRouter(t, nuevoFakeRekognitionAutorizado())
	token := crearNegocio(t, router, "Taqueria")

	empRec := doJSON(t, router, token, http.MethodPost, "/empleados", map[string]interface{}{"nombre_completo": "Juan Perez"})
	var empleado struct {
		ID int `json:"id"`
	}
	decode(t, empRec, &empleado)

	rostroJuan := base64.StdEncoding.EncodeToString([]byte("bytes-de-la-cara-de-juan"))
	rostroDesconocido := base64.StdEncoding.EncodeToString([]byte("bytes-de-una-cara-distinta"))

	// Enrolar: el "SDK" simulado indexa el rostro y devuelve un FaceId.
	enrolarRec := doJSON(t, router, token, http.MethodPost, fmt.Sprintf("/empleados/%d/enrolar", empleado.ID), map[string]interface{}{
		"imagen": rostroJuan,
	})
	if enrolarRec.Code != http.StatusOK {
		t.Fatalf("enrolar: esperaba 200, obtuve %d (%s)", enrolarRec.Code, enrolarRec.Body.String())
	}
	var enrolado struct {
		RekognitionFaceID string `json:"rekognition_face_id"`
	}
	decode(t, enrolarRec, &enrolado)
	if enrolado.RekognitionFaceID == "" {
		t.Fatalf("enrolar no devolvió rekognition_face_id: %s", enrolarRec.Body.String())
	}

	// Marcar con la MISMA imagen: el "SDK" simulado debe reconocer a Juan y
	// registrarle una entrada.
	marcarRec := doJSON(t, router, token, http.MethodPost, "/asistencia/marcar", map[string]interface{}{"imagen": rostroJuan})
	if marcarRec.Code != http.StatusOK {
		t.Fatalf("marcar (rostro conocido): esperaba 200, obtuve %d (%s)", marcarRec.Code, marcarRec.Body.String())
	}
	var reconocido struct {
		EmpleadoID int    `json:"empleado_id"`
		Tipo       string `json:"tipo"`
	}
	decode(t, marcarRec, &reconocido)
	if reconocido.EmpleadoID != empleado.ID || reconocido.Tipo != "entrada" {
		t.Fatalf("marcar reconoció mal al empleado: %+v", reconocido)
	}

	// Marcar con una imagen DISTINTA: nadie en la colección coincide, debe
	// dar 404 "rostro no reconocido" (no 503 — Rekognition sí contestó).
	noReconocidoRec := doJSON(t, router, token, http.MethodPost, "/asistencia/marcar", map[string]interface{}{"imagen": rostroDesconocido})
	if noReconocidoRec.Code != http.StatusNotFound {
		t.Fatalf("marcar (rostro desconocido): esperaba 404, obtuve %d (%s)", noReconocidoRec.Code, noReconocidoRec.Body.String())
	}
}

// TestFallaCuandoAwsNoAutoriza simula que AWS rechaza las credenciales
// (UnrecognizedClientException, el mismo error que se ve en los logs reales
// cuando faltan o son inválidas) y comprueba que el sistema responde de
// forma distinta a un "rostro no reconocido": 503 (servicio no disponible),
// no 404 — para que el kiosco le diga a RH "revisa AWS" en vez de sugerir
// que el empleado no está registrado.
func TestFallaCuandoAwsNoAutoriza(t *testing.T) {
	router, _ := setupRouter(t, nuevoFakeRekognitionSinAutorizacion())
	token := crearNegocio(t, router, "Taqueria")

	empRec := doJSON(t, router, token, http.MethodPost, "/empleados", map[string]interface{}{"nombre_completo": "Empleado Nuevo"})
	var empleado struct {
		ID int `json:"id"`
	}
	decode(t, empRec, &empleado)

	imagen := base64.StdEncoding.EncodeToString([]byte("cualquier-imagen"))

	enrolarRec := doJSON(t, router, token, http.MethodPost, fmt.Sprintf("/empleados/%d/enrolar", empleado.ID), map[string]interface{}{
		"imagen": imagen,
	})
	if enrolarRec.Code != http.StatusServiceUnavailable {
		t.Errorf("enrolar sin autorización de AWS: esperaba 503, obtuve %d (%s)", enrolarRec.Code, enrolarRec.Body.String())
	}

	marcarRec := doJSON(t, router, token, http.MethodPost, "/asistencia/marcar", map[string]interface{}{"imagen": imagen})
	if marcarRec.Code != http.StatusServiceUnavailable {
		t.Errorf("marcar sin autorización de AWS: esperaba 503 (no 404, para no confundirlo con 'rostro no reconocido'), obtuve %d (%s)", marcarRec.Code, marcarRec.Body.String())
	}

	// El registro manual (SPEC.md sección 8) no depende de Rekognition y
	// debe seguir funcionando aunque AWS esté caído.
	manualRec := doJSON(t, router, token, http.MethodPost, "/asistencia/marcar-manual", map[string]interface{}{
		"empleado_id": empleado.ID,
		"tipo":        "entrada",
	})
	if manualRec.Code != http.StatusOK {
		t.Errorf("marcar-manual sin AWS: esperaba 200, obtuve %d (%s)", manualRec.Code, manualRec.Body.String())
	}
}

// TestPerfilKioscoSoloAsistencia es la prueba central de los perfiles: una
// tablet logueada como "kiosco" solo puede marcar asistencia (facial,
// manual, ver "hoy" y listar empleados para el selector del registro
// manual) — todo lo demás (empleados, turnos, reportes, usuarios) es
// exclusivo de "admin".
func TestPerfilKioscoSoloAsistencia(t *testing.T) {
	router, _ := setupRouter(t, nuevoFakeRekognitionAutorizado())
	adminToken := crearNegocio(t, router, "Taqueria")

	empRec := doJSON(t, router, adminToken, http.MethodPost, "/empleados", map[string]interface{}{"nombre_completo": "Empleado Kiosco"})
	var empleado struct {
		ID int `json:"id"`
	}
	decode(t, empRec, &empleado)

	// El admin da de alta la cuenta de la tablet.
	kioscoRec := doJSON(t, router, adminToken, http.MethodPost, "/usuarios/registro", map[string]interface{}{
		"username": "tablet_mostrador",
		"password": "clave-de-la-tablet",
		"rol":      "kiosco",
	})
	if kioscoRec.Code != http.StatusCreated {
		t.Fatalf("crear usuario kiosco: esperaba 201, obtuve %d (%s)", kioscoRec.Code, kioscoRec.Body.String())
	}

	// Un rol inválido se rechaza antes de tocar la base de datos.
	rolInvalidoRec := doJSON(t, router, adminToken, http.MethodPost, "/usuarios/registro", map[string]interface{}{
		"username": "otro", "password": "x", "rol": "superadmin",
	})
	if rolInvalidoRec.Code != http.StatusBadRequest {
		t.Errorf("rol inválido: esperaba 400, obtuve %d (%s)", rolInvalidoRec.Code, rolInvalidoRec.Body.String())
	}

	loginRec := doJSON(t, router, "", http.MethodPost, "/login", map[string]interface{}{
		"username": "tablet_mostrador", "password": "clave-de-la-tablet",
	})
	var loginResp struct {
		Token string `json:"token"`
		Rol   string `json:"rol"`
	}
	decode(t, loginRec, &loginResp)
	if loginResp.Rol != "kiosco" || loginResp.Token == "" {
		t.Fatalf("login de la tablet no devolvió el rol/token esperado: %+v", loginResp)
	}
	kioscoToken := loginResp.Token

	// Permitido para "kiosco": marcar (facial y manual), ver hoy, y listar
	// empleados (lo usa el selector del registro manual).
	permitido := []struct {
		method, path string
		body         interface{}
	}{
		{http.MethodGet, "/empleados", nil},
		{http.MethodGet, "/asistencia/hoy", nil},
		{http.MethodPost, "/asistencia/marcar-manual", map[string]interface{}{"empleado_id": empleado.ID, "tipo": "entrada"}},
		{http.MethodPost, "/asistencia/marcar", map[string]interface{}{"imagen": base64.StdEncoding.EncodeToString([]byte("rostro-cualquiera"))}},
	}
	for _, caso := range permitido {
		rec := doJSON(t, router, kioscoToken, caso.method, caso.path, caso.body)
		if rec.Code == http.StatusForbidden {
			t.Errorf("kiosco: %s %s debería estar permitido, dio 403 (%s)", caso.method, caso.path, rec.Body.String())
		}
	}

	// Prohibido para "kiosco": todo lo administrativo.
	prohibido := []struct {
		method, path string
		body         interface{}
	}{
		{http.MethodPost, "/empleados", map[string]interface{}{"nombre_completo": "Intento"}},
		{http.MethodPatch, fmt.Sprintf("/empleados/%d/estado", empleado.ID), map[string]interface{}{"estado": "inactivo"}},
		{http.MethodPost, fmt.Sprintf("/empleados/%d/enrolar", empleado.ID), map[string]interface{}{"imagen": "x"}},
		{http.MethodPost, "/turnos", map[string]interface{}{"nombre": "x", "hora_entrada": "08:00:00", "hora_salida": "17:00:00"}},
		{http.MethodGet, "/turnos", nil},
		{http.MethodGet, "/reportes/asistencia", nil},
		{http.MethodPost, "/usuarios/registro", map[string]interface{}{"username": "otro2", "password": "x"}},
		{http.MethodGet, "/usuarios", nil},
	}
	for _, caso := range prohibido {
		rec := doJSON(t, router, kioscoToken, caso.method, caso.path, caso.body)
		if rec.Code != http.StatusForbidden {
			t.Errorf("kiosco: %s %s debería dar 403, dio %d (%s)", caso.method, caso.path, rec.Code, rec.Body.String())
		}
	}

	// El admin sí puede ver la lista de usuarios del negocio, incluida la tablet.
	usuariosRec := doJSON(t, router, adminToken, http.MethodGet, "/usuarios", nil)
	var usuarios []struct {
		Username string `json:"username"`
		Rol      string `json:"rol"`
	}
	decode(t, usuariosRec, &usuarios)
	encontrado := false
	for _, u := range usuarios {
		if u.Username == "tablet_mostrador" && u.Rol == "kiosco" {
			encontrado = true
		}
	}
	if !encontrado {
		t.Fatalf("el admin no vio la cuenta de la tablet en /usuarios: %+v", usuarios)
	}
}
