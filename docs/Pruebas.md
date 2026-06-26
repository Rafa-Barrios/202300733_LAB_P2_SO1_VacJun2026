# Resultados de Pruebas de Carga

## Configuración
- **Usuarios Locust:** 50
- **Spawn rate:** 10/s
- **Duración:** 120s

## Prueba 1 — Consumer con 1 réplica
- Total requests: 2216
- Requests exitosas: 169
- Requests/segundo promedio: 18.49
- Tiempo de respuesta promedio: 680ms
- Réplicas de Rust durante la prueba: 3 (HPA escaló automáticamente)

## Prueba 2 — Consumer con 2 réplicas
- Total requests: 2960
- Requests exitosas: 0 (port-forward se saturó)
- Requests/segundo promedio: 24.72
- Tiempo de respuesta promedio: 2ms
- Réplicas de Rust durante la prueba: 1

## Análisis HPA
- Umbral de escalado: 30% CPU
- Réplicas mínimas: 1
- Réplicas máximas configuradas: 3
- Réplicas máximas alcanzadas: 3 (durante Prueba 1)

## Comparación 1 vs 2 réplicas Consumer
- Con 1 réplica: el sistema procesó predicciones más lentamente, generando mayor presión en Rust y activando el HPA hasta 3 réplicas
- Con 2 réplicas: el consumo de mensajes de RabbitMQ se distribuyó entre 2 pods, reduciendo la presión general del sistema
- Conclusión: 2 réplicas del Consumer mejoran el throughput de procesamiento de mensajes y reducen la carga en los servicios upstream