package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	ufoV1 "github.com/yyunoshev/yyunoshev_go/week_1/grpc/pkg/proto/ufo/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const grpcPort = 50051

type ufoService struct {
	ufoV1.UnimplementedUFOServiceServer // Мы копируем все методы интерфейса и будем их сами переопределять

	mu        sync.RWMutex
	sightings map[string]*ufoV1.Sighting
}

func (s *ufoService) Create(_ context.Context, req *ufoV1.CreateRequest) (*ufoV1.CreateResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	newUUID := uuid.NewString()
	sighting := &ufoV1.Sighting{
		Uuid:      newUUID,
		Info:      req.GetInfo(),
		CreatedAt: timestamppb.New(time.Now()),
	}

	s.sightings[newUUID] = sighting
	log.Printf("Create new ufo with uuid: %s", newUUID)
	return &ufoV1.CreateResponse{
		Uuid: newUUID,
	}, nil
}

func (s *ufoService) Get(_ context.Context, req *ufoV1.GetRequest) (*ufoV1.GetResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sighting, ok := s.sightings[req.GetUuid()]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "sighing with UUID %s not found", req.GetUuid())
	}

	return &ufoV1.GetResponse{
		Sighting: sighting,
	}, nil
}

func (s *ufoService) Update(_ context.Context, req *ufoV1.UpdateRequest) (*emptypb.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sighting, ok := s.sightings[req.GetUuid()]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "sighing with UUID %s not found", req.GetUuid())
	}
	if req.UpdateInfo == nil {
		return nil, status.Errorf(codes.InvalidArgument, "UpdateInfo is nil")
	}

	if req.GetUpdateInfo().ObservedAt != nil {
		sighting.Info.ObservedAt = req.UpdateInfo.GetObservedAt()
	}

	if req.GetUpdateInfo().Location != nil {
		sighting.Info.Location = req.UpdateInfo.Location.Value
	}

	if req.GetUpdateInfo().Description != nil {
		sighting.Info.Description = req.GetUpdateInfo().Description.Value
	}

	if req.GetUpdateInfo().Color != nil {
		sighting.Info.Color = req.GetUpdateInfo().Color
	}

	if req.GetUpdateInfo().Sound != nil {
		sighting.Info.Sound = req.GetUpdateInfo().Sound
	}

	if req.GetUpdateInfo().DurationSeconds != nil {
		sighting.Info.DurationSeconds = req.GetUpdateInfo().DurationSeconds
	}

	sighting.UpdatedAt = timestamppb.New(time.Now())
	return &emptypb.Empty{}, nil
}

func (s *ufoService) Delete(_ context.Context, req *ufoV1.DeleteRequest) (*emptypb.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sighting, ok := s.sightings[req.GetUuid()]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "sighing with UUID %s not found", req.GetUuid())
	}

	sighting.DeletedAt = timestamppb.New(time.Now())
	return &emptypb.Empty{}, nil
}

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		log.Printf("Failed to listen: %v\n", err)
		return
	}
	defer func() {
		if cerr := lis.Close(); cerr != nil {
			log.Printf("Failed to close listener: %v\n", cerr)
		}
	}()

	s := grpc.NewServer()

	service := &ufoService{
		sightings: make(map[string]*ufoV1.Sighting),
	}

	ufoV1.RegisterUFOServiceServer(s, service)

	// Рефлексия - это возможность клиента спрашивать какие есть методы у сервера
	// из-за этого в постмане можно сразу увидеть список методов
	reflection.Register(s)

	go func() {
		log.Printf("🚀 Starting gRPC server on port %d\n", grpcPort)
		err = s.Serve(lis)
		if err != nil {
			log.Printf("Failed to serve: %v\n", err)
			return
		}
	}()
	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🛑 Shutting down gRPC server...")
	s.GracefulStop()
	log.Println("✅ Server stopped")
}
