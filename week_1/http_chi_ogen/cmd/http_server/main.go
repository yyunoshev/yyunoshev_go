package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	customMiddleware "github.com/yyunoshev/yyunoshev_go/week_1/http_chi_ogen/internal/middleware"
	weatherV1 "github.com/yyunoshev/yyunoshev_go/week_1/http_chi_ogen/pkg/openapi/weather/v1"
)

const (
	httpPort          = "8089"
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 10 * time.Second

	defaultErrorCode = http.StatusInternalServerError
)

type WeatherStorage struct {
	mu       sync.RWMutex
	weathers map[string]*weatherV1.Weather
}

func NewWeatherStorage() *WeatherStorage {
	return &WeatherStorage{
		weathers: make(map[string]*weatherV1.Weather),
	}
}

func (s *WeatherStorage) GetWeather(city string) *weatherV1.Weather {
	s.mu.RLock()
	defer s.mu.RUnlock()

	weather, ok := s.weathers[city]
	if !ok {
		return nil
	}
	return weather
}

func (s *WeatherStorage) UpdateWeather(city string, weather *weatherV1.Weather) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.weathers[city] = weather
}

type WeatherHandler struct {
	storage *WeatherStorage
}

func NewWeatherHandler(storage *WeatherStorage) *WeatherHandler {
	return &WeatherHandler{
		storage: storage,
	}
}

// TODO:Почему не используем контекст?
func (h *WeatherHandler) GetWeatherByCity(_ context.Context, params weatherV1.GetWeatherByCityParams) (weatherV1.GetWeatherByCityRes, error) {
	weather := h.storage.GetWeather(params.City)
	if weather == nil {
		return &weatherV1.NotFoundError{
			Code:    404,
			Message: fmt.Sprintf("Weather for city '%s' not found", params.City),
		}, nil
	}
	return weather, nil
}

func (h *WeatherHandler) UpdateWeatherByCity(_ context.Context, req *weatherV1.UpdateWeatherRequest, params weatherV1.UpdateWeatherByCityParams) (weatherV1.UpdateWeatherByCityRes, error) {
	weather := &weatherV1.Weather{
		City:        params.City,
		Temperature: req.Temperature,
		UpdatedAt:   time.Now(),
	}

	h.storage.UpdateWeather(params.City, weather)
	return weather, nil
}

func (h *WeatherHandler) NewError(_ context.Context, err error) *weatherV1.GenericErrorStatusCode {
	return &weatherV1.GenericErrorStatusCode{
		StatusCode: defaultErrorCode,
		Response: weatherV1.GenericError{
			Code:    weatherV1.NewOptInt(defaultErrorCode),
			Message: weatherV1.NewOptString(err.Error()),
		},
	}
}

func main() {
	storage := NewWeatherStorage()

	weatherHandler := NewWeatherHandler(storage)

	weatherServer, err := weatherV1.NewServer(weatherHandler)
	if err != nil {
		log.Fatalf("Ошибка в создании сервера OpenAPI: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(readHeaderTimeout))
	r.Use(customMiddleware.RequestLogger)

	r.Mount("/", weatherServer)

	server := &http.Server{
		Addr:    net.JoinHostPort("localhost", httpPort),
		Handler: r,
		// Защита от Slowloris атак - тип DDoS-атаки, при которой
		// атакующий умышленно медленно отправляет HTTP-заголовки, удерживая соединения открытыми и истощая
		// пул доступных соединений на сервере. ReadHeaderTimeout принудительно закрывает соединение,
		// если клиент не успел отправить все заголовки за отведенное время.
		ReadTimeout: readHeaderTimeout,
	}

	go func() {
		log.Printf("🚀 HTTP-сервер запущен на порту %s\n", httpPort)
		err = server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("❌ Ошибка запуска сервера: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Завершение работы сервера")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	err = server.Shutdown(ctx)
	if err != nil {
		log.Printf("❌ Ошибка при остановке сервера: %v\n", err)
	}

	log.Println("✅ Сервер остановлен")
}
