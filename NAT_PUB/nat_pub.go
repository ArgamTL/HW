package main

import (
	"fmt"
	"io/ioutil"
	"os"

	//"strconv"

	"github.com/nats-io/stan.go"
)

func main() {

	/* Conection should be established before running this Go script */
	st, err := stan.Connect("test-cluster", "2")
	if err != nil {
		panic(err.Error())
	}
	defer st.Close()

	// Open our jsonFile
	jsonFile, err := os.Open("model.json")
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Successfully Opened model.json")
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	// read our opened xmlFile as a byte array.
	byteValue, _ := ioutil.ReadAll(jsonFile) // []byte("Hello World")
	if err := st.Publish("test", byteValue); err != nil {
		panic(err.Error())
	}

	////////////////////TESTING///////////////////////////////

	/*
		var result map[string]interface{}

		if err := json.Unmarshal(byteValue, &result); err != nil {
			panic(err.Error())
		}

		fmt.Println(result["items"])

		//////////END OF TESTIN//////////////////////////////////
	*/
}
