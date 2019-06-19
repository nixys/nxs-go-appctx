package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

const (
	requestMaxPOSTBodySize = 512
)

type resp struct {
	Data string `json:"data"`
}

type reqPOST struct {
	Data string `json:"data"`
}

func epRoutesSet(ctx selfContext) *mux.Router {

	r := mux.NewRouter()

	r.Handle("/action1/{args1}", epAction1(ctx)).Methods("GET")
	r.Handle("/action2", epAction2(ctx)).Methods("POST")
	r.Handle("/action3", epAction3(ctx)).Methods("POST")

	return r
}

func epAction1(ctx selfContext) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		log.Debugln("ctx.timeInterval [action1]:", ctx.conf.Bind)

		vars := mux.Vars(r)
		args := vars["args1"]

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")

		epSendResponse(w, r, http.StatusOK, args)
		return
	})
}

func epAction2(ctx selfContext) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		post := reqPOST{}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")

		// Set request max body size
		r.Body = http.MaxBytesReader(w, r.Body, requestMaxPOSTBodySize)

		if r.ContentLength > requestMaxPOSTBodySize {
			epSendResponse(w, r, http.StatusRequestEntityTooLarge, "")
			return
		}

		// Retrieve json from body
		if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
			epSendResponse(w, r, http.StatusBadRequest, "")
			return
		}

		epSendResponse(w, r, http.StatusOK, post.Data)
		return
	})
}

func epAction3(ctx selfContext) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")

		// Set request max body size
		r.Body = http.MaxBytesReader(w, r.Body, requestMaxPOSTBodySize)

		if r.ContentLength > requestMaxPOSTBodySize {
			epSendResponse(w, r, http.StatusRequestEntityTooLarge, "")
			return
		}

		v := r.FormValue("test_form")

		epSendResponse(w, r, http.StatusOK, v)
		return
	})
}

func epSendResponse(w http.ResponseWriter, r *http.Request, httpCode int, data string) {

	var rStatus resp

	w.WriteHeader(httpCode)

	if httpCode == http.StatusOK {
		rStatus.Data = data
	} else {
		rStatus.Data = "error"
	}

	if err := json.NewEncoder(w).Encode(rStatus); err != nil {
		log.WithFields(logrus.Fields{
			"remote_addr": r.RemoteAddr,
			"metod":       r.Method,
			"url":         r.URL,
			"code":        httpCode,
		}).Errorf("response send error: %v", err)
	}

	log.Infof("%s \"%s %s\" %d", r.RemoteAddr, r.Method, r.URL, httpCode)
}
