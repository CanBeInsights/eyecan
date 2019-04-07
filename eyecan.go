package main

import (
	"fmt"
	"github.com/bclicn/color"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/minerva-eyecan/eyecan/hex"
	"github.com/minerva-eyecan/eyecan/watson"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
)

var PORT string

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	prefix := log.Prefix()
	log.SetPrefix(color.Blue(prefix))

	if runtime.GOOS == "linux" {
		PORT = "80"
	} else {
		PORT = "8081"
	}
}

// logHandler wraps a request handler in logging to mark the start and end of a request, and logs errors
func logHandler(f func(w http.ResponseWriter,req *http.Request) error) func(w http.ResponseWriter,req *http.Request) {
	return func(w http.ResponseWriter,req *http.Request) {
		reqHash := context.Get(req, "hash")
		log.Println(reqHash, color.Green("Starting request processing"))
		err := f(w, req)
		if err != nil {
			log.Println(reqHash, color.Red("Error on request:"), err.Error())
		}
		log.Println(reqHash, color.Green("End of request"))
	}
}

func allowCORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqHash := context.Get(r, "hash")

		//w.Header().Set("Content-Type", "application/json; charset=utf-8")
		//w.Header().Set("Access-Control-Allow-Origin", "*")

		w.Header().Set("Access-Control-Allow-Origin", "*");
		w.Header().Set("Access-Control-Allow-Credentials", "true");
		w.Header().Set("Access-Control-Allow-Methods", "GET,HEAD,OPTIONS,POST,PUT");
		w.Header().Set("Access-Control-Allow-Headers", "Access-Control-Allow-Headers, Origin,Accept, X-Requested-With, Content-Type, Access-Control-Request-Method, Access-Control-Request-Headers");

		log.Println(reqHash, "Adding CORS headers")

		next.ServeHTTP(w, r)
	})
}

func requestHashMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hash := hex.GetRand(12)
		context.Set(r, "hash", hash)

		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hash := context.Get(r, "hash")
		ipForwardedFor := r.Header.Get("X-Forwarded-For")
		ipRemoteAddr := r.RemoteAddr
		var ipString string
		if ipForwardedFor != "" {
			ipString = ipForwardedFor + " (X-Forwarded-For)"
		} else if ipRemoteAddr != "" {
			ipString = ipRemoteAddr + " (RemoteAddr)"
		} else {
			ipString = "unknown"
		}
		log.Println(hash, color.Green("New request from"), color.Green(ipString),
			color.Green("for"), color.Green(r.RequestURI))

		next.ServeHTTP(w, r)
	})
}

func InformUserOnGet(w http.ResponseWriter, req *http.Request) error {
	_, err := w.Write([]byte("`POST` your data as `application/JSON` to /extract"))
	if err != nil {
		return err
	}

	return nil
}

func ExtractCategories(w http.ResponseWriter, req *http.Request) error {
	hash := context.Get(req, "hash")

	// Hopefully handle `OPTIONS` requests correctly
	if (*req).Method == "OPTIONS" {
		log.Println(hash, "Responded to OPTIONS request")
		return nil
	}

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}

	//params := mux.Vars(req)
	//var user User
	//_= json.NewDecoder(req.Body).Decode(&user)
	//user.ID=params["id"]
	//users=append(users,user)

	//input := req.Body
	output, watsonErr := watson.LookupsCategories(string(body))
	if watsonErr != nil {
		return watsonErr
	}

	// TODO: Find a better way of isolating `err`s from each other than this
	_, writeErr := w.Write([]byte(output))
	if writeErr != nil {
		return writeErr
	}

	return nil
}

func main() {

	fmt.Println("Starting up on port", PORT)
	//data := watson.LookupCategories("https://cloud.ibm.com/apidocs/natural-language-understanding?code=go")
	//_ = watson.LookupsCategories(input)
	//fmt.Println(data)

	router := mux.NewRouter()
	router.Use(allowCORSMiddleware)
	router.Use(requestHashMiddleware)
	router.Use(loggingMiddleware)

	router.HandleFunc("/extract", logHandler(InformUserOnGet)).Methods("GET")
	router.HandleFunc("/extract", logHandler(ExtractCategories)).Methods("POST",  "OPTIONS")
	//router.HandleFunc("/users", logHandler(GetUsers)).Methods("GET")
	//router.HandleFunc("/users/{id}", logHandler(GetUserById)).Methods("GET")
	//router.HandleFunc("/users/{id}", logHandler(CreateUser)).Methods("POST")
	//router.HandleFunc("/users/{id}", logHandler(DeleteUser)).Methods("DELETE")

	log.Fatal(http.ListenAndServe(":" + PORT, router))
}
