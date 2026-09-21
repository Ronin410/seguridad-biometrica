package rekognition

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
)

// Client envuelve el SDK de AWS Rekognition contra una colección de rostros
// dedicada a empleados (separada de la colección de GuarderiaBiometric, ver
// SPEC.md sección 3).
type Client struct {
	sdk          *rekognition.Client
	collectionID string
}

func NewClient(ctx context.Context, region, collectionID string) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("cargando configuración de AWS: %w", err)
	}

	return &Client{
		sdk:          rekognition.NewFromConfig(cfg),
		collectionID: collectionID,
	}, nil
}

// EnsureCollection crea la colección de empleados si todavía no existe.
func (c *Client) EnsureCollection(ctx context.Context) error {
	_, err := c.sdk.CreateCollection(ctx, &rekognition.CreateCollectionInput{
		CollectionId: aws.String(c.collectionID),
	})
	if err != nil {
		var alreadyExists *types.ResourceAlreadyExistsException
		if errors.As(err, &alreadyExists) {
			return nil
		}
		return fmt.Errorf("creando colección de Rekognition: %w", err)
	}
	return nil
}

// IndexFace registra el rostro de un empleado y devuelve el FaceId asignado
// por Rekognition (rekognition_face_id en el modelo de datos).
func (c *Client) IndexFace(ctx context.Context, imageBytes []byte, externalImageID string) (string, error) {
	result, err := c.sdk.IndexFaces(ctx, &rekognition.IndexFacesInput{
		CollectionId:    aws.String(c.collectionID),
		Image:           &types.Image{Bytes: imageBytes},
		ExternalImageId: aws.String(externalImageID),
		MaxFaces:        aws.Int32(1),
	})
	if err != nil {
		return "", fmt.Errorf("indexando rostro: %w", err)
	}
	if len(result.FaceRecords) == 0 {
		return "", fmt.Errorf("no se detectó ningún rostro en la imagen")
	}

	return *result.FaceRecords[0].Face.FaceId, nil
}

// SearchFace busca coincidencias del rostro capturado contra la colección de
// empleados y devuelve el FaceId y la similitud del mejor resultado.
func (c *Client) SearchFace(ctx context.Context, imageBytes []byte, similarityThreshold float32) (faceID string, similarity float64, err error) {
	result, err := c.sdk.SearchFacesByImage(ctx, &rekognition.SearchFacesByImageInput{
		CollectionId:       aws.String(c.collectionID),
		Image:              &types.Image{Bytes: imageBytes},
		FaceMatchThreshold: aws.Float32(similarityThreshold),
		MaxFaces:           aws.Int32(1),
	})
	if err != nil {
		return "", 0, fmt.Errorf("buscando rostro: %w", err)
	}
	if len(result.FaceMatches) == 0 {
		return "", 0, fmt.Errorf("rostro no reconocido")
	}

	match := result.FaceMatches[0]
	return *match.Face.FaceId, float64(*match.Similarity), nil
}
