package internalapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
	"io"

	api  "go-parkinsons-server/internal/api/gen"
	stub "go-parkinsons-server/gen-stubs"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"google.golang.org/grpc"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

var _ api.ServerInterface = (*RPCHandler)(nil)

type RPCHandler struct {
	client stub.AudioStreamingClient
}

func NewRPCHandler(grpcAddr string) (*RPCHandler, error) {
	conn, err := grpc.Dial(grpcAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	log.Printf("[gRPC] connected to %s", grpcAddr)
	return &RPCHandler{client: stub.NewAudioStreamingClient(conn)}, nil
}

func (h *RPCHandler) DetectWs(c echo.Context, params api.DetectWsParams) error {
	age := params.Age
	sex := params.Sex
	log.Printf("[WS] upgrade request age=%d sex=%d", age, sex)

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("[WS] upgrade FAILED: %v", err)
		return err
	}
	defer conn.Close()
	log.Printf("[WS] upgrade OK")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	stream, err := h.client.DetectParkinsonsFromAudio(ctx)
	if err != nil {
		log.Printf("[gRPC] stream init FAILED: %v", err)
		conn.WriteJSON(map[string]string{"error": "grpc stream init failed"})
		return err
	}
	log.Printf("[gRPC] stream opened")

	if err := stream.Send(&stub.AudioChunks{
		IsMetadata: true,
		Age:        int32(age),
		Sex:        int32(sex),
	}); err != nil {
		log.Printf("[gRPC] metadata send FAILED: %v", err)
		conn.WriteJSON(map[string]string{"error": "grpc metadata send failed"})
		return err
	}
	log.Printf("[gRPC] metadata sent age=%d sex=%d", age, sex)

	accumulator := make([]byte, 0, 1280)
	chunkID     := int32(1)
	totalBytes  := 0
	done        := false

	for !done {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway) {
				log.Printf("[WS] closed normally — flushing")
				done = true
			} else {
				log.Printf("[WS] read ERROR: %v", err)
				done = true
			}
			break
		}

		switch msgType {
		case websocket.TextMessage:
			var ctrl map[string]string
			if json.Unmarshal(msg, &ctrl) == nil && ctrl["type"] == "done" {
				log.Printf("[WS] done sentinel — total=%d bytes chunks=%d",
					totalBytes, chunkID-1)
				done = true
			}

		case websocket.BinaryMessage:
			totalBytes += len(msg)
			accumulator = append(accumulator, msg...)
			log.Printf("[WS] binary +%d bytes (total=%d accum=%d)",
				len(msg), totalBytes, len(accumulator))

			for len(accumulator) >= 640 {
				if sendErr := stream.Send(&stub.AudioChunks{
					RawAudioChunk: accumulator[:640],
					ChunkId:       chunkID,
				}); sendErr != nil {
					log.Printf("[gRPC] send FAILED chunk=%d: %v", chunkID, sendErr)
					if closeErr := stream.CloseSend(); closeErr != nil {
						log.Printf("[gRPC] CloseSend error: %v", closeErr)
					}
					conn.WriteJSON(map[string]string{"error": "grpc send failed"})
					return sendErr
				}
				log.Printf("[gRPC] sent chunk=%d (640 bytes)", chunkID)
				accumulator = accumulator[640:]
				chunkID++
			}
		}
	}

	if len(accumulator) > 0 {
		log.Printf("[gRPC] flushing tail %d bytes as chunk=%d", len(accumulator), chunkID)
		if sendErr := stream.Send(&stub.AudioChunks{
			RawAudioChunk: accumulator,
			ChunkId:       chunkID,
		}); sendErr != nil {
			log.Printf("[gRPC] tail flush FAILED: %v", sendErr)
		} else {
			chunkID++
		}
	}

	log.Printf("[gRPC] CloseAndRecv — waiting for Python... total_chunks=%d total_bytes=%d",
		chunkID-1, totalBytes)
	resp, err := stream.CloseAndRecv()
	if err != nil {
		log.Printf("[gRPC] recv FAILED: %v", err)
		conn.WriteJSON(map[string]string{"error": "grpc recv failed"})
		return err
	}
	log.Printf("[gRPC] result isHavingParkinsons=%v severity=%.3f",
		resp.IsHavingParkinsons, resp.Severity)

	var voiceFeatures map[string]interface{}
	if resp.ExtractedVoiceFeatures != nil {
		voiceFeatures = resp.ExtractedVoiceFeatures.AsMap()
	}

	result := map[string]interface{}{
		"isHavingParkinsons":     resp.IsHavingParkinsons,
		"severity":               resp.Severity,
		"suggestion":             resp.Suggestion,
		"extractedVoiceFeatures": voiceFeatures,
	}

	log.Printf("[WS] sending result back to FE")
	return conn.WriteJSON(result)
}

func (h *RPCHandler) DetectUpload(c echo.Context, params api.DetectUploadParams) error {
    file, err := c.FormFile("audio")
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "audio file required")
    }
    src, err := file.Open()
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "cannot open file")
    }
    defer src.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
    defer cancel()

    stream, err := h.client.DetectParkinsonsFromAudio(ctx)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "grpc stream init failed")
    }

    // Send metadata first — same pattern as WS handler
    stream.Send(&stub.AudioChunks{
        IsMetadata: true,
        Age:        int32(params.Age),
        Sex:        int32(params.Sex),
    })

    // Stream file in 640-byte chunks — same chunk size as WS handler
    buf := make([]byte, 640)
    chunkID := int32(1)
    for {
        n, err := src.Read(buf)
        if n > 0 {
            stream.Send(&stub.AudioChunks{
                RawAudioChunk: buf[:n],
                ChunkId:       chunkID,
            })
            chunkID++
        }
        if err == io.EOF {
            break
        }
        if err != nil {
            return echo.NewHTTPError(http.StatusInternalServerError, "file read error")
        }
    }

    resp, err := stream.CloseAndRecv()
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "grpc recv failed")
    }

    var voiceFeatures map[string]interface{}
    if resp.ExtractedVoiceFeatures != nil {
        voiceFeatures = resp.ExtractedVoiceFeatures.AsMap()
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
        "isHavingParkinsons":     resp.IsHavingParkinsons,
        "severity":               resp.Severity,
        "suggestion":             resp.Suggestion,
        "extractedVoiceFeatures": voiceFeatures,
    })
}