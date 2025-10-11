package middleware

import (
	"log"
	"net/http"
	"time"
)

// RequestLogger создает middleware для логирования времени выполнения запросов
func RequestLogger(next http.Handler) http.Handler {
	// TODO: next - What is this?
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		log.Printf("⏱️ Начало запроса: %s %s", r.Method, r.URL.Path)
		// Передаем управление следующему обработчику
		next.ServeHTTP(w, r)

		duration := time.Since(startTime)
		log.Printf("✅ Запрос завершен: %s %s, время выполнения: %v", r.Method, r.URL.Path, duration)

	})
}
