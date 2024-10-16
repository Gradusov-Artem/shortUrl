package main

// импорты
import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kataras/iris/v12"
	"log"
	"strconv"
	"time"
)

// структура запроса
type RequestBody struct {
	URL string `json:"original_url"`
}

// структура ответа
type ResponseBody struct {
	URL1 string `json:"original_url"`
	URL2 string `json:"short_url"`
}

/*
Функция main.
Создает соединение с базой данных.
Запускает сервер.
*/
func main() {
	db, err := connect_to_the_database()
	if err != nil {
		log.Fatal("Ошибка подключения к базе данных:", err)
	}
	defer db.Close()

	app := iris.New()

	app.Get("/{short_url}", func(ctx iris.Context) {
		get_from_short_url(ctx, db)
	})

	app.Post("/short", func(ctx iris.Context) {
		get_short_url(ctx, db)
	})

	app.Run(iris.Addr(":8080"))
}

/*
Функция для получения original_url из short_url.
Предварительно осуществляется проверка на существование и актуальность ссылки.
Доступен по /<short_url>, <short_url> - сокращенная ссылка.
*/
func get_from_short_url(ctx iris.Context, db *pgxpool.Pool) {
	short_url := ctx.Params().Get("short_url")
	var original_url string
	err := db.QueryRow(context.Background(), "SELECT original_url FROM urls WHERE short_url = $1 AND creation_date >= NOW() - INTERVAL '86400 seconds'", short_url).Scan(&original_url)
	if err != nil {
		ctx.JSON(iris.Map{
			"message": "Возможно ссылка устарела, проверьте правильность ввода или создайте новую!",
		})
		return
	}

	ctx.Redirect(original_url, iris.StatusFound)
	return
}

/*
Функция для создания short_url из original_url.
Предварительно осуществляется проверка на существование ссылки в базе данных.
Доступен по /short, в body содержится url.
*/
func get_short_url(ctx iris.Context, db *pgxpool.Pool) {
	var requestBody RequestBody

	if err := ctx.ReadJSON(&requestBody); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(iris.Map{"error": err.Error()})
		return
	}

	var existingUrl string
	err := db.QueryRow(context.Background(), "SELECT short_url FROM urls WHERE original_url = $1", requestBody.URL).Scan(&existingUrl)

	if err == nil {
		ctx.JSON(ResponseBody{
			URL1: requestBody.URL,
			URL2: "http://" + ctx.Request().Host + "/" + existingUrl,
		})
		return
	}

	var short_url string
	short_url = generate_short_url()

	_, err = db.Exec(context.Background(), "INSERT INTO urls (original_url, short_url, creation_date) VALUES ($1, $2, CURRENT_TIMESTAMP)", requestBody.URL, short_url)
	if err != nil {
		// логирование ошибки
		log.Printf("Ошибка при вставке в базу данных: %v\n", err)
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.JSON(iris.Map{"error": "Ошибка при сохранении данных в базе данных"})
		return
	}

	response := ResponseBody{
		URL1: requestBody.URL,
		URL2: "http://" + ctx.Request().Host + "/" + short_url,
	}

	ctx.JSON(response)
}

/*
Функция для создания подключения к базе данных.
*/
func connect_to_the_database() (*pgxpool.Pool, error) {
	db_string := "host=localhost user=postgres password=7719150Artik dbname=shortLink port=5432"
	pool, err := pgxpool.New(context.Background(), db_string)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

/*
Функция для создания сокращенной ссылки используя время UNIX.
*/
func generate_short_url() string {
	var short_url = time.Now().Unix()
	return strconv.FormatInt(short_url, 10)
}
