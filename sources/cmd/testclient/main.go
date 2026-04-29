package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	pb "github.com/rozoomcool/sihkaromicro/sources/gen/proto/sources"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const (
	chunkSize  = 1024 * 1024 // 1MB chunks
	serverAddr = "localhost:50053"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Usage: go run main.go <file_path> <token>")
	}

	filePath := os.Args[1]
	token := os.Args[2]

	// Подключаемся
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("failed to connect:", err)
	}
	defer conn.Close()

	client := pb.NewSourcesServiceClient(conn)

	// Добавляем токен
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.Pairs("authorization", "Bearer "+token),
	)

	// Открываем стрим
	stream, err := client.UploadSource(ctx)
	if err != nil {
		log.Fatal("failed to open stream:", err)
	}

	// Открываем файл
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal("failed to open file:", err)
	}
	defer file.Close()

	stat, _ := file.Stat()

	// 1. Отправляем метаданные
	err = stream.Send(&pb.UploadSourceRequest{
		Data: &pb.UploadSourceRequest_Meta{
			Meta: &pb.SourceMeta{
				ProjectId: 1,
				Name:      stat.Name(),
				Type:      pb.SourceType_SOURCE_TYPE_PDF,
				Size:      stat.Size(),
			},
		},
	})
	if err != nil {
		log.Fatal("failed to send meta:", err)
	}
	fmt.Println("✅ Metadata sent")

	// 2. Стримим чанки
	buf := make([]byte, chunkSize)
	totalSent := 0

	for {
		n, err := file.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("failed to read file:", err)
		}

		if err := stream.Send(&pb.UploadSourceRequest{
			Data: &pb.UploadSourceRequest_Chunk{
				Chunk: buf[:n],
			},
		}); err != nil {
			log.Fatal("failed to send chunk:", err)
		}

		totalSent += n
		fmt.Printf("📤 Sent %d / %d bytes\n", totalSent, stat.Size())
	}

	// 3. Говорим done
	if err := stream.Send(&pb.UploadSourceRequest{
		Data: &pb.UploadSourceRequest_Done{
			Done: true,
		},
	}); err != nil {
		log.Fatal("failed to send done:", err)
	}
	fmt.Println("✅ Done signal sent")

	// 4. Получаем ответ
	resp, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatal("failed to receive response:", err)
	}

	fmt.Printf("✅ Upload successful!\n")
	fmt.Printf("   source_id: %d\n", resp.SourceId)
	fmt.Printf("   job_id:    %s\n", resp.JobId)
}
