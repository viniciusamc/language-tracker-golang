package main

import (
	"fmt"
	"net/http"
)

func userCreate(w http.ResponseWriter, r *http.Request){
	fmt.Fprint(w, "hello world")
}


