package api

import (
	"net/http"
)

type Databases struct {
	DbBroker DbBroker
}

func (d Databases) Create(w http.ResponseWriter, req *http.Request) {
	
}

func (d Databases) Get(w http.ResponseWriter, req *http.Request) {

}

func (d Databases) Update(w http.ResponseWriter, req *http.Request) {

}

func (d Databases) Delete(w http.ResponseWriter, req *http.Request) {

}
