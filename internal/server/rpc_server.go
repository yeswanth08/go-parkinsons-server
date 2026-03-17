package internalapi

import (
	"context"
	"io"
	"log"
	"net/http"
	"time"

	api "go-parkinsons-server/internal/api/gen"
	stub "go-parkinsons-server/gen-stubs"

	"github.com/labstack/echo/v4"
	"google.golang.org/grpc"
)

type RPCHandler struct {
	client stub.AudioStreamingClient
}

func NewRPCHandler(grpcAddr string) (*RPCHandler, error) {
	conn, err := grpc.Dial(grpcAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return &RPCHandler{client: stub.NewAudioStreamingClient(conn)}, nil
}

func (h *RPCHandler) PostApiV1Detect(c echo.Context, params api.PostApiV1DetectParams) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()

	age := params.Age
	sex := params.Sex

	stream, err := h.client.DetectParkinsonsFromAudio(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "stream init failed"})
	}

	// send metadata as first chunk
	if err := stream.Send(&stub.AudioChunks{
		IsMetadata: true,
		Age:        int32(age),
		Sex:        int32(sex),
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "metadata send failed"})
	}

	queue := make([]byte, 0, 640*100)
	tmp := make([]byte, 4096)
	chunkId := int32(1)

	for {
		n, readErr := c.Request().Body.Read(tmp)
		if n > 0 {
			queue = append(queue, tmp[:n]...)
			for len(queue) >= 640 {
				if sendErr := stream.Send(&stub.AudioChunks{
					RawAudioChunk: queue[:640],
					ChunkId:       chunkId,
				}); sendErr != nil {
					log.Printf("grpc send failed chunk %d: %v", chunkId, sendErr)
				    if closeErr := stream.CloseSend(); closeErr != nil {
        				log.Printf("grpc close send failed: %v", closeErr)
    				}
					return c.JSON(http.StatusInternalServerError, map[string]string{"error": "stream send failed"})
				}
				queue = queue[640:]
				chunkId++
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			if closeErr := stream.CloseSend(); closeErr != nil {
        		log.Printf("grpc close send failed: %v", closeErr)
    		}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "body read failed"})
		}
	}

	// flush remaining bytes as final partial chunk
	if len(queue) > 0 {
		if closeErr := stream.CloseSend(); closeErr != nil {
        	log.Printf("grpc close send failed: %v", closeErr)
    	}
		stream.Send(&stub.AudioChunks{RawAudioChunk: queue, ChunkId: chunkId})
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "grpc recv failed"})
	}

	isHaving := resp.IsHavingParkinsons
	severity  := resp.Severity
	suggestion := resp.Suggestion
	voiceFeatures := resp.ExtractedVoiceFeatures.AsMap()

	return c.JSON(http.StatusOK, api.ParkinsonsResponse{
		IsHavingParkinsons: &isHaving,
		Severity:           &severity,
		Suggestion:         &suggestion,
		ExtracedVoiceFeatures: &voiceFeatures,
	})
}