#[macro_use]
extern crate rocket;

// импорт модулей
use rocket::serde::{json::Json, Deserialize, Serialize};
use rocket_db_pools::{Database, Connection};
use deadpool_postgres::{self, Client, Pool, Config};
use deadpool_postgres::tokio_postgres::NoTls;
use rocket::response::Redirect;
use std::time::{SystemTime, UNIX_EPOCH};
use serde::de::Unexpected::Option;

// структуры
#[derive(Database)] // объявляем базу данных
#[database("url_db")]
struct UrlDatabase(Pool);

#[derive(Serialize, Deserialize)]
struct Urls {
    short_url: String,
    original_url: String,
}

#[derive(Serialize, Deserialize)]
struct Url {
    original_url: String
}

/**
Функция для вывода сообщения об ошибке
Доступна по /timeout.
*/
#[get("/timeout")]
async fn write_timeout_error() -> &'static str {
    "Возможно ссылка устарела, проверьте правильность ввода или создайте новую!"
}

/**
Функция для получения original_url из short_url
Предварительно осуществляется проверка на существование и актуальность ссылки
Доступен по /<short_url>, <short_url> - сокращенная ссылка.
*/
#[get("/<short_url>")]
async fn get_from_short_url(db: Connection<UrlDatabase>, short_url: String) ->  Redirect{
    let client: &Client = &*db;

    // проверка на существование original_url и то, что она не просрочена
    let query = "SELECT original_url FROM urls WHERE short_url = $1 AND creation_date >= NOW() - INTERVAL '86400 seconds'";
    let row = client.query_opt(query, &[&short_url]).await.unwrap();

    if let Some(row) = row {
        let original_url: String = row.get(0);
        Redirect::to(original_url)
    }
    else {
        Redirect::to("http://localhost:8000/timeout")
    }
}

/**
Функция для создания short_url из original_url
Предварительно осуществляется проверка на существование ссылки в базе данных
Доступен по /short, в body содержится url.
 */
#[post("/short", format = "json", data = "<url>")]
async fn get_short_url(db: Connection<UrlDatabase>, url: Json<Url>) -> Json<Urls> {
    let client: &Client = &*db;
    let mut short_url: String;

    // проверка на существование в базе данных ссылки на данный ресурс
    let query = "SELECT short_url FROM urls WHERE original_url = $1";
    let row = client.query_opt(query, &[&url.original_url]).await;

    if let Ok(Some(row)) = row {
        short_url = row.get(0);
    }
    else {
        short_url = generate_short_url();
        let query = "INSERT INTO urls (original_url, short_url, creation_date) VALUES ($1, $2, CURRENT_TIMESTAMP)";
        client
            .execute(query, &[&url.original_url, &short_url])
            .await
            .unwrap();
    }
    Json(Urls {
        original_url: (&*url.original_url).to_string(),
        short_url: format!("http://localhost:8000/{}", short_url),
    })
}

/**
Функция main
Осуществляет создание config, connection pool и запуск сервера.
*/
#[rocket::main]
async fn main() -> Result<(), rocket::Error> {
    let mut cfg = Config::new();
    cfg.dbname = Some("shortLink".to_string());
    cfg.user = Some("postgres".to_string());
    cfg.password = Some("7719150Artik".to_string());
    cfg.host = Some("localhost".to_string());
    cfg.port = Some(5432);

    let pool = cfg.create_pool(None, NoTls).unwrap();

    rocket::build()
        .manage(UrlDatabase(pool))
        .mount("/", routes![get_short_url, get_from_short_url, write_timeout_error])
        .launch()
        .await?;
    Ok(())
}

/**
Функция для создания сокращенной ссылки используя время UNIX.
*/
fn generate_short_url() -> String {
    let short_url: String = SystemTime::now().duration_since(UNIX_EPOCH).expect("").as_secs().to_string();
    short_url
}
