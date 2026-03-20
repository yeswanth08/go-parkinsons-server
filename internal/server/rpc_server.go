package internalapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	api  "go-parkinsons-server/internal/api/gen"
	stub "go-parkinsons-server/gen-stubs"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"google.golang.org/grpc"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

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

func (h *RPCHandler) DetectWs(c echo.Context, params api.DetectWsParams) error {
	age := params.Age
	sex := params.Sex
	log.Printf("[WS] new connection age=%d sex=%d", age, sex)

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("[WS] upgrade failed: %v", err)
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(c.Request().Context(), 60*time.Second)
	defer cancel()

	stream, err := h.client.DetectParkinsonsFromAudio(ctx)
	if err != nil {
		log.Printf("[gRPC] stream init failed: %v", err)
		conn.WriteJSON(map[string]string{"error": "grpc stream init failed"})
		return err
	}

	if err := stream.Send(&stub.AudioChunks{
		IsMetadata: true,
		Age:        int32(age),
		Sex:        int32(sex),
	}); err != nil {
		log.Printf("[gRPC] metadata send failed: %v", err)
		conn.WriteJSON(map[string]string{"error": "grpc metadata send failed"})
		return err
	}
	log.Printf("[gRPC] metadata sent")

	accumulator := make([]byte, 0, 1280)
	chunkID     := int32(1)
	totalBytes  := 0

	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway) {
				log.Printf("[WS] normal close, total=%d bytes", totalBytes)
			} else {
				log.Printf("[WS] read error: %v", err)
			}
			break
		}

		if msgType == websocket.TextMessage {
			var ctrl map[string]string
			if json.Unmarshal(msg, &ctrl) == nil && ctrl["type"] == "done" {
				log.Printf("[WS] done sentinel received, total=%d bytes", totalBytes)
				break
			}
			continue
		}

		totalBytes += len(msg)
		accumulator = append(accumulator, msg...)
		log.Printf("[WS] recv %d bytes (total=%d)", len(msg), totalBytes)

		for len(accumulator) >= 640 {
			if sendErr := stream.Send(&stub.AudioChunks{
				RawAudioChunk: accumulator[:640],
				ChunkId:       chunkID,
			}); sendErr != nil {
				log.Printf("[gRPC] send failed chunk=%d: %v", chunkID, sendErr)
				stream.CloseSend()
				conn.WriteJSON(map[string]string{"error": "grpc send failed"})
				return sendErr
			}
			accumulator = accumulator[640:]
			chunkID++
		}
	}

	if len(accumulator) > 0 {
		if sendErr := stream.Send(&stub.AudioChunks{
			RawAudioChunk: accumulator,
			ChunkId:       chunkID,
		}); sendErr != nil {
			log.Printf("[gRPC] tail flush failed: %v", sendErr)
		} else {
			log.Printf("[gRPC] flushed %d tail bytes", len(accumulator))
		}
	}

	log.Printf("[gRPC] waiting for response…")
	resp, err := stream.CloseAndRecv()
	if err != nil {
		log.Printf("[gRPC] recv failed: %v", err)
		conn.WriteJSON(map[string]string{"error": "grpc recv failed"})
		return err
	}
	log.Printf("[gRPC] result isHavingParkinsons=%v severity=%.3f",
		resp.IsHavingParkinsons, resp.Severity)

	var voiceFeatures map[string]interface{}
	if resp.ExtractedVoiceFeatures != nil {
		voiceFeatures = resp.ExtractedVoiceFeatures.AsMap()
	}

	return conn.WriteJSON(map[string]interface{}{
		"isHavingParkinsons":     resp.IsHavingParkinsons,
		"severity":               resp.Severity,
		"suggestion":             resp.Suggestion,
		"extractedVoiceFeatures": voiceFeatures,
	})
}