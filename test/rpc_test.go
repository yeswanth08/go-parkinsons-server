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
	// let's stream the basic raw audio file to python service using rpc and get result
	// go-lang will be the client service to get the reponse from the python -server
	// conn, err :=  grpc.NewClient("localhost:50051")

	//  syntax depricated but will be revamp later :(

	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect the py-rpc server %v", err)
	}
	defer conn.Close()
	
	// crete a client for the service
	client := stub.NewAudioStreamingClient(conn)
	
	// getting the ctx with background ctx
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// calling the remote procedure/function
	// as this is an client stream to server msg we needed to create a stream first and then the result
	stream, err := client.DetectParkinsonsFromAudio(ctx)

	if err != nil {
		log.Fatal(err)
	}

	//  this is not ideal streaming of the audio chunks i.e we are streaming with one chunk
	// audioFile, err := os.ReadFile("./healthy/temp.wav")

	// imp using audio stream by chunk fragmentation
	// fileStats, statErr := os.Stat("./healthy/temp.wav")
	audioFile, ioErr := os.Open("./parkinsons/temp.wav")

	// if statErr != nil {
	// 	fmt.Print("error in opening the audio file")
	// 	return
	// }

	if ioErr != nil {
		// fmt.Print("error in opening the audio file")
		// return
		// insted of this we can simply use the print err + os.exit = fatal
		log.Fatal(err)
	}
	
	defer audioFile.Close()

	// skipping the wav header
	_, err = audioFile.Seek(44, io.SeekStart)
	if err != nil {
		log.Fatal(err)
	}

	audioBuffer := make([]byte, 640) 
	// ideal chunk size 
	chunkId := int32(1);

	for {
		audioChunk, err := audioFile.Read(audioBuffer)
		if err == io.EOF {
			// if we reached the end of the file
			break; 
		}
		if err != nil {
			// print + os.exit
			log.Fatal(err)
		}

		// now we created a stream now from that stream send the chunks
		err = stream.Send(&stub.AudioChunks{
			RawAudioChunk: audioBuffer[:audioChunk],
			ChunkId:       chunkId,
		})
		if err != nil {
			log.Fatal(err)
		}
		chunkId++;
	}

	// close the stream and get the response
	resp, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("cannot get request %v", err)
	}

	fmt.Printf("resp %v", resp.IsHavingParkinsons)
}
