package main

import (
	//"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"

	//"encoding/gob"
	"encoding/json"
	"fmt"

	//"os"

	"time"

	//"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/jellydator/ttlcache/v2"
	_ "github.com/lib/pq"
	"github.com/nats-io/stan.go"
)

type Order struct {
	Order_uid          string `json: "order_uid"`
	Track_number       string `json: "track_number"`
	Entry              string `json: "entry,omitempty"`
	Delivery           Delivery
	Payment            Payment
	Items              Items
	Locale             string `json: "locale,omitempty"`
	Internal_signature string `json: "internal_signature,omitempty"`
	Customer_id        string `json: "customer_id,omitempty"`
	Delivery_service   string `json: "sdelivery_service,omitempty"`
	Shardkey           string `json: "shardkey,omitempty"`
	Sm_id              int    `json: "sm_id,omitempty"`
	Date_created       string `json: "date_created,omitempty"`
	Oof_shard          string `json: "oof_shard,omitempty"`
}

type Item struct {
	Chrt_id      int    `json: "chrt_id"`
	Track_number string `json: "track_number,omitempty"`
	Price        int    `json: "price,omitempty"`
	Rid          string `json: "rid,omitempty"`
	Name         string `json: "name,omitempty"`
	Sale         int    `json: "sale,omitempty"`
	Size         string `json: "size,omitempty"`
	Total_price  int    `json: "total_price"`
	Nm_id        int    `json: "nm_id"`
	Brand        string `json: "brand,omitempty"`
	Status       int    `json: "status,omitempty"`
}

type Items []Item

type Payment struct {
	Transaction   string `json: "transaction"`
	Request_id    string `json: "request_id",omitempty"`
	Currency      string `json: "currency"`
	Provider      string `json: "provider,omitempty"`
	Amount        int    `json: "amount,omitempty"`
	Payment_dt    int64  `json: "payment_dt,omitempty"`
	Bank          string `json: "bank,omitempty"`
	Delivery_cost int    `json: "delivery_cost"`
	Goods_total   int    `json: "goods_total"`
	Custom_fee    int    `json: "custom_fee"`
}

type Delivery struct {
	Name    string `json: "name,omitempty"`
	Phone   string `json: "phone,omitempty"`
	Zip     string `json: "zip,omitempty"`
	City    string `json: "city,omitempty"`
	Address string `json: "address,omitempty"`
	Region  string `json: "region,omitempty"`
	Email   string `json: "email,omitempty"`
}

var cache ttlcache.SimpleCache = ttlcache.NewCache()

func (a Order) Value() (driver.Value, error) {
	return json.Marshal(a)
}
func (a *Order) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
}

// /////////////////////////DB CONNECTION CONSTANTS///////////////////////
const (
	host     = "localhost"
	port     = 5432
	user     = "user"
	password = "your_pass"
	dbname   = "your_db"
)

func main() {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	fmt.Println("Successfully connected! (Creating the  \"Orders\" table [IF NOT EXISTS])")

	query := `CREATE TABLE IF NOT EXISTS Orders (
		order_id SERIAL PRIMARY KEY,
		order_uid character(19) UNIQUE,
		orders JSONB)`

	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()

	res, err := db.ExecContext(ctx, query)
	if err != nil {
		fmt.Printf("Error %s when creating 'Orders' table", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		fmt.Printf("Error %s when getting rows affected", err)
	}
	fmt.Printf("Rows affected when creating table: %d \n", rows)

	/* Connecting Nats|stan :
	conection should be established before running this Go script */
	st, err := stan.Connect("test-cluster", "2")
	if err != nil {
		panic(err.Error())
	}
	defer st.Close()

	/* Subscribing to Nats */
	var result_byte []byte
	received_order := Order{}

	sub, err := st.Subscribe("test", func(m *stan.Msg) {
		if json.Valid(m.Data) {
			result_byte = m.Data
			if err := json.Unmarshal(result_byte, &received_order); err != nil {
				panic(err.Error())
			}
		} else {
			fmt.Println("error: INVALID JSONB!")
			return
		}
	}, stan.StartWithLastReceived(),
	)
	if err != nil {
		panic(err.Error())
	}

	// Unsubscribe
	sub.Unsubscribe()

	// Close connection
	st.Close()

	sqlStatement_insrt := `INSERT INTO Orders (orders, order_uid) VALUES ($1,$2);`
	_, err = db.Exec(sqlStatement_insrt, result_byte, received_order.Order_uid)
	if err != nil {
		//log.Fatal(err)
		//panic(err)
		fmt.Println(err.Error())
	}

	//////////////////TTLcaching/////////////////////////

	cache.SetTTL(time.Duration(5 * time.Hour))
	cache.Set(received_order.Order_uid, received_order)

	//////////////////////Fiber///////////////////////////

	app := fiber.New()

	app.Get("/:order_uid", func(c *fiber.Ctx) error {
		order_uid_ := c.Params("order_uid")
		val, err := cache.Get(order_uid_)
		if err != nil {
			the_order := new(Order)
			sqlStatement_slqt_row := `SELECT orders FROM Orders WHERE order_uid = $1;`
			err = db.QueryRow(sqlStatement_slqt_row, order_uid_).Scan(&the_order)

			if err != nil {
				panic(err)
			}
			return c.JSON(the_order)
		}
		return c.JSON(val)
	})
	app.Listen(":3000")
}
