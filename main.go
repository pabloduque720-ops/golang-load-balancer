package main

import (
	"fmt"
	"net/http"

	"haproxy-manager/handlers"
	"haproxy-manager/utils"
)

func balancersRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handlers.GetBalancers(w, r)
	case http.MethodPost:
		handlers.CreateBalancer(w, r)
	case http.MethodPut:
		handlers.UpdateBalancer(w, r)
	case http.MethodDelete:
		handlers.DeleteBalancer(w, r)
	default:
		utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func serversRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handlers.GetServers(w, r)
	case http.MethodPost:
		handlers.CreateServer(w, r)
	case http.MethodPut:
		handlers.UpdateServer(w, r)
	case http.MethodDelete:
		handlers.DeleteServer(w, r)
	default:
		utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func applyHAProxyRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	handlers.ApplyFullHAProxyConfig(w, r)
}

func main() {
	http.HandleFunc("/balancers", balancersRoute)
	http.HandleFunc("/servers", serversRoute)
	http.HandleFunc("/haproxy/apply", applyHAProxyRoute)

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	fmt.Println("Servidor iniciado en http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}