// basic test suit desgined for streaming the audio chunks via grpc
package test

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"testing"
	"time"

	stub "go-parkinsons-server/gen-stubs"
	"google.golang.org/grpc"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestDetectParkinsons(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect the py-rpc server %v", err)
	}
	defer conn.Close()

	client := stub.NewAudioStreamingClient(conn)

	tests := []struct {
		name     string
		audioFile string
		age      int32
		sex      int32
		wantParkinsons bool
	}{
		{
			name:           "healthy sample",
			audioFile:      "./healthy/temp.wav",
			age:            65,
			sex:            0,
			wantParkinsons: false,
		},
		{
			name:           "parkinsons sample",
			audioFile:      "./parkinsons/temp.wav",
			age:            65,
			sex:            0,
			wantParkinsons: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := streamAudioFile(t, client, tt.audioFile, tt.age, tt.sex)
			fmt.Printf("[%s] isHavingParkinsons=%v severity=%.4f\n",
				tt.name, result.IsHavingParkinsons, result.Severity)

			if result.IsHavingParkinsons != tt.wantParkinsons {
				t.Errorf("expected isHavingParkinsons=%v got=%v",
					tt.wantParkinsons, result.IsHavingParkinsons)
			}
		})
	}
}

func streamAudioFile(t *testing.T, client stub.AudioStreamingClient, filePath string, age, sex int32) *stub.ParkinsonsDetectionResult {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// calling the remote procedure/function
	// as this is an client stream to server msg we needed to create a stream first and then the result
	stream, err := client.DetectParkinsonsFromAudio(ctx)

	if err != nil {
		t.Fatalf("stream init failed: %v", err)
	}

	// send metadata as first chunk
	if err := stream.Send(&stub.AudioChunks{
		IsMetadata: true,
		Age:        age,
		Sex:        sex,
	}); err != nil {
		t.Fatalf("metadata send failed: %v", err)
	}

	// open and stream wav file
	f, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("failed to open audio file: %v", err)
	}
	defer f.Close()

	// skip WAV header
	if _, err := f.Seek(44, io.SeekStart); err != nil {
		t.Fatalf("failed to seek past wav header: %v", err)
	}

	buf := make([]byte, 640)
	chunkId := int32(1)

	for {
		n, err := f.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read failed at chunk %d: %v", chunkId, err)
		}
		if sendErr := stream.Send(&stub.AudioChunks{
			RawAudioChunk: buf[:n],
			ChunkId:       chunkId,
		}); sendErr != nil {
			t.Fatalf("send failed at chunk %d: %v", chunkId, sendErr)
		}
		chunkId++
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		t.Fatalf("recv failed: %v", err)
	}

	log.Printf("chunks sent: %d", chunkId-1)
	return resp
}
