package coretelemetry

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	DependencyDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "dependency_duration_seconds",
			Help:    "Время выполнения запросов к внешним зависимостям",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"dependency", "operation"},
	)

	//core:kafka:StateReplicator
	SyncEventsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_sync_events_processed_total",
			Help: "Общее количество обработанных событий в StateReplicator",
		},
		[]string{"event_type", "status"},
	)
	//core:kafka:StateReplicator:end

	//core:kafka:OutboxRelay
	OutboxRelayedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_outbox_relayed_total",
			Help: "Общее количество отправленных сообщений из Outbox",
		},
	)

	EventDeliveryDelay = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kafka_event_delivery_delay_seconds",
			Help:    "Время от генерации события (Outbox) до его финальной обработки (Lag)",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 5, 10, 30, 60, 120, 300},
		},
		[]string{"event_type"},
	)
	//core:kafka:OutboxRelay:end
	//core:kafka:TaskProcessor
	TasksProcessedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_tasks_processed_total",
			Help: "Общее количество выполненных фоновых задач (TaskProcessor)",
		},
		[]string{"task_type", "status"},
	)
	//core:kafka:TaskProcessor:end
)

func InitMetrics(serviceName string, enabled bool) {
	if !enabled {
		return
	}

	// Создаем умный регистратор, который ко всем метрикам добавит {service="имя_из_конфига"}
	reg := prometheus.WrapRegistererWith(
		prometheus.Labels{"service": serviceName},
		prometheus.DefaultRegisterer,
	)

	// Регистрируем метрики ядра через этот обернутый регистратор
	reg.MustRegister(
		DependencyDuration,
		//core:kafka:StateReplicator
		SyncEventsTotal,
		//core:kafka:StateReplicator:end
		//core:kafka:OutboxRelay
		OutboxRelayedTotal,
		EventDeliveryDelay,
		//core:kafka:OutboxRelay:end
		//core:kafka:TaskProcessor
		TasksProcessedTotal,
		//core:kafka:TaskProcessor:end
	)
}
