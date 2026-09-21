package handlers

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"staffattendance/internal/rekognition"
)

// fakeRekognition simula AWS Rekognition sin credenciales ni red: en vez de
// comparar rostros de verdad, considera "el mismo rostro" a dos llamadas con
// exactamente los mismos bytes de imagen (suficiente para probar el flujo de
// enrolamiento/reconocimiento) y expone un modo "sin autorización" para
// simular que AWS rechaza las credenciales — el mismo error real que se ve
// en los logs cuando faltan o son inválidas
// (UnrecognizedClientException).
type fakeRekognition struct {
	mu          sync.Mutex
	autorizado  bool
	colecciones map[string]map[string]string // collectionID -> hash de la imagen -> faceID
	contador    int
}

var _ rekognition.FaceRecognizer = (*fakeRekognition)(nil)

func nuevoFakeRekognitionAutorizado() *fakeRekognition {
	return &fakeRekognition{autorizado: true, colecciones: map[string]map[string]string{}}
}

func nuevoFakeRekognitionSinAutorizacion() *fakeRekognition {
	return &fakeRekognition{autorizado: false}
}

var errAutorizacionAWSSimulada = errors.New("simulado: UnrecognizedClientException: el token de seguridad incluido en la solicitud es inválido")

func (f *fakeRekognition) EnsureCollection(ctx context.Context, collectionID string) error {
	if !f.autorizado {
		return errAutorizacionAWSSimulada
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.colecciones[collectionID] == nil {
		f.colecciones[collectionID] = map[string]string{}
	}
	return nil
}

func (f *fakeRekognition) IndexFace(ctx context.Context, collectionID string, imageBytes []byte, externalImageID string) (string, error) {
	if !f.autorizado {
		return "", errAutorizacionAWSSimulada
	}
	if len(imageBytes) == 0 {
		return "", rekognition.ErrRostroNoDetectado
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.contador++
	faceID := fmt.Sprintf("fake-face-%d", f.contador)
	if f.colecciones[collectionID] == nil {
		f.colecciones[collectionID] = map[string]string{}
	}
	f.colecciones[collectionID][string(imageBytes)] = faceID
	return faceID, nil
}

func (f *fakeRekognition) SearchFace(ctx context.Context, collectionID string, imageBytes []byte, similarityThreshold float32) (string, float64, error) {
	if !f.autorizado {
		return "", 0, errAutorizacionAWSSimulada
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	faceID, ok := f.colecciones[collectionID][string(imageBytes)]
	if !ok {
		return "", 0, rekognition.ErrRostroNoReconocido
	}
	return faceID, 99.9, nil
}
