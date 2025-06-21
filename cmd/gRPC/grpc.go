package main

import (
	"context"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"log"
	"net"
	"net/http"
	"sync"

	"bikeDomain/internal/metric"
	desc "bikeDomain/pkg/gRPC"
)

const grpcPort = 50051
const address = ":50051"

type server struct {
	desc.UnimplementedBikeServer
	mu        sync.RWMutex
	userBikes map[uint64]map[uint64]*desc.BikeInfo // userID -> bikeID Хранилище
	IdCounter uint64                               // Будет считать id аля как в бд
}

func NewBikeServer() *server {
	return &server{
		userBikes: make(map[uint64]map[uint64]*desc.BikeInfo),
		IdCounter: 1,
	}
}

func (s *server) GetBikeByUser(ctx context.Context, req *desc.GetBikeByUserRequest) (*desc.GetBikeByUserResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userID := req.GetUserId()
	bikesMap, exists := s.userBikes[userID]
	if !exists {
		return &desc.GetBikeByUserResponse{
			Bikes: []*desc.BikeInfo{},
		}, nil
	}

	// конвертируем map в slice
	bikes := make([]*desc.BikeInfo, 0, len(bikesMap))
	for _, bike := range bikesMap {
		bikes = append(bikes, bike)
	}

	return &desc.GetBikeByUserResponse{Bikes: bikes}, nil
}

func (s *server) CreateBikeByUser(ctx context.Context, req *desc.CreateBikeByUserRequest) (*desc.CreateBikeByUserResponse, error) {

	s.mu.Lock()
	defer s.mu.Unlock()

	userID := req.GetUserId()
	bikeID := s.IdCounter
	s.IdCounter++

	if _, exists := s.userBikes[userID]; !exists {
		s.userBikes[userID] = make(map[uint64]*desc.BikeInfo)
	}

	if _, exists := s.userBikes[userID]; !exists {
		s.userBikes[userID] = make(map[uint64]*desc.BikeInfo)
	}

	bike := &desc.BikeInfo{
		BikeId: bikeID,
		Model:  req.GetModel(),
		Color:  req.GetColor(),
		IsWork: req.GetIsWork(),
	}

	s.userBikes[userID][bikeID] = bike
	s.IdCounter++

	log.Printf("Created bike: ID=%d, Model=%s", bike.BikeId, bike.Model)
	return &desc.CreateBikeByUserResponse{BikeId: bikeID}, nil
}

func (s *server) UpdateBikeByUser(ctx context.Context, req *desc.UpdateBikeByUserRequest) (*desc.UpdateBikeByUserResponse, error) {

	// Валидация
	if req.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "User ID is required")
	}
	if req.GetBikeId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "Bike ID is required")
	}
	if req.GetModel() == "" {
		return nil, status.Error(codes.InvalidArgument, "Model is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	userID := req.GetUserId()
	bikeID := req.GetBikeId()

	// Проверяем наличие велосипеда
	bikes, userExists := s.userBikes[userID]
	if !userExists {
		return nil, status.Errorf(codes.NotFound, "User %d not found", userID)
	}

	bike, bikeExists := bikes[bikeID]
	if !bikeExists {
		return nil, status.Errorf(codes.NotFound, "Bike %d not found", bikeID)
	}

	// Обновляем данные
	bike.Model = req.GetModel()
	bike.Color = req.GetColor()
	bike.IsWork = req.GetIsWork()

	log.Printf("Updated bike: UserID=%d, BikeID=%d", userID, bikeID)
	return &desc.UpdateBikeByUserResponse{Success: true}, nil
}

func (s *server) DeleteBikeByUser(ctx context.Context, req *desc.DeleteBikeByUserRequest) (*desc.DeleteBikeByUserResponse, error) {
	// Валидация
	if req.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "User ID is required")
	}
	if req.GetBikeId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "Bike ID is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	userID := req.GetUserId()
	bikeID := req.GetBikeId()

	// Проверяем существование пользователя
	bikes, userExists := s.userBikes[userID]
	if !userExists {
		return nil, status.Errorf(codes.NotFound, "User %d not found", userID)
	}

	// Проверяем существование велосипеда
	if _, bikeExists := bikes[bikeID]; !bikeExists {
		return nil, status.Errorf(codes.NotFound, "Bike %d not found", bikeID)
	}

	// Удаляем
	delete(bikes, bikeID)

	// Если у пользователя больше нет велосипедов - удаляем запись
	if len(bikes) == 0 {
		delete(s.userBikes, userID)
	}

	log.Printf("Deleted bike: UserID=%d, BikeID=%d", userID, bikeID)
	return &desc.DeleteBikeByUserResponse{Success: true}, nil
}

func main() {
	ctx := context.Background()
	err := metric.Init(ctx)
	if err != nil {
		log.Fatalf("failed to init metrics: %v", err)
	}

	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer(
		grpc.UnaryInterceptor(
			metric.MetricsInterceptor(),
		),
	)
	reflection.Register(s)
	bikeServer := NewBikeServer()
	desc.RegisterBikeServer(s, bikeServer)

	log.Printf("server listening at %v", lis.Addr())

	go func() {
		err = runPrometheus()
		if err != nil {
			log.Fatal(err)
		}
	}()

	if err = s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func runPrometheus() error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	prometheusServer := &http.Server{
		Addr:    "localhost:2112",
		Handler: mux,
	}

	log.Printf("Prometheus server is running on %s", "localhost:2112")

	err := prometheusServer.ListenAndServe()
	if err != nil {
		return err
	}

	return nil
}
