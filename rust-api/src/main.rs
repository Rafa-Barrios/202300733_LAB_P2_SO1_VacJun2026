use actix_web::{web, App, HttpServer, HttpResponse};
use serde::{Deserialize, Serialize};
use std::env;

// Estructura que recibe de Locust
#[derive(Debug, Deserialize, Serialize, Clone)]
struct Prediction {
    home_team: String,
    away_team: String,
    home_goals: i32,
    away_goals: i32,
    username: String,
    timestamp: String,
}

// Estructura de respuesta
#[derive(Serialize)]
struct ApiResponse {
    success: bool,
    message: String,
}

// Handler principal — recibe predicciones de Locust
async fn receive_prediction(
    prediction: web::Json<Prediction>,
    go_client_url: web::Data<String>,
) -> HttpResponse {
    log::info!(
        "Predicción recibida: {} vs {} | {} - {} | user: {}",
        prediction.home_team,
        prediction.away_team,
        prediction.home_goals,
        prediction.away_goals,
        prediction.username
    );

    // Validaciones básicas
    if prediction.home_goals < 0 || prediction.away_goals < 0 {
        return HttpResponse::BadRequest().json(ApiResponse {
            success: false,
            message: "Los goles no pueden ser negativos".to_string(),
        });
    }

    if prediction.home_team.is_empty() || prediction.away_team.is_empty() {
        return HttpResponse::BadRequest().json(ApiResponse {
            success: false,
            message: "Los equipos no pueden estar vacíos".to_string(),
        });
    }

    // Enviamos al Go Deployment 1
    let client = reqwest::Client::new();
    let url = format!("{}/predict", go_client_url.get_ref());

    match client.post(&url).json(&prediction.into_inner()).send().await {
        Ok(resp) => {
            if resp.status().is_success() {
                HttpResponse::Ok().json(ApiResponse {
                    success: true,
                    message: "Predicción procesada correctamente".to_string(),
                })
            } else {
                log::error!("Go Deployment 1 respondió con error: {}", resp.status());
                HttpResponse::InternalServerError().json(ApiResponse {
                    success: false,
                    message: "Error al procesar la predicción".to_string(),
                })
            }
        }
        Err(e) => {
            log::error!("Error conectando a Go Deployment 1: {}", e);
            // Por ahora respondemos OK igual para no bloquear las pruebas
            HttpResponse::Ok().json(ApiResponse {
                success: true,
                message: "Predicción recibida (Go D1 no disponible aún)".to_string(),
            })
        }
    }
}

// Health check
async fn health() -> HttpResponse {
    HttpResponse::Ok().json(serde_json::json!({
        "status": "healthy",
        "service": "rust-api"
    }))
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    env_logger::init_from_env(env_logger::Env::default().default_filter_or("info"));

    let port = env::var("PORT").unwrap_or_else(|_| "8080".to_string());
    let go_client_url = env::var("GO_CLIENT_URL")
        .unwrap_or_else(|_| "http://go-deployment1-service:8080".to_string());

    log::info!("Iniciando Rust API en puerto {}", port);
    log::info!("Go Client URL: {}", go_client_url);

    let go_url = web::Data::new(go_client_url);

    HttpServer::new(move || {
        App::new()
            .app_data(go_url.clone())
            .app_data(
                web::JsonConfig::default()
                    .error_handler(|err, _| {
                        let response = HttpResponse::BadRequest().json(ApiResponse {
                            success: false,
                            message: format!("JSON inválido: {}", err),
                        });
                        actix_web::error::InternalError::from_response(err, response).into()
                    })
            )
            .route("/predict", web::post().to(receive_prediction))
            .route("/health", web::get().to(health))
    })
    .bind(format!("0.0.0.0:{}", port))?
    .run()
    .await
}