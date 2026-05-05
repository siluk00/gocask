package main

import (
	"fmt"
	"log"
	"os"

	"github.com/siluk00/gocask"
)

func main() {
	dir := "./tmp_data"
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	db, err := gocask.Open(gocask.Config{Dir: dir})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	key := []byte("hello")
	value := []byte("world")

	fmt.Printf("Putting %s=%s\n", key, value)
	err = db.Put(key, value)
	if err != nil {
		log.Fatal(err)
	}

	val, err := db.Get(key)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Got %s=%s\n", key, val)

	fmt.Println("Deleting key")
	err = db.Delete(key)
	if err != nil {
		log.Fatal(err)
	}

	val, err = db.Get(key)
	if err == nil {
		log.Fatalf("expected error getting deleted key, got %s", val)
	}
	fmt.Println("Key successfully deleted (not found)")
}
