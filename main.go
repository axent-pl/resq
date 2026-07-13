package main

import (
	"log"
	"net/http"
	"os"

	"axent.pl/resq/handler"
	"axent.pl/resq/middleware"
	"axent.pl/resq/model"
	"axent.pl/resq/router"
	"axent.pl/resq/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	if err := db.AutoMigrate(&model.Event{}, &model.EventParticipant{}, &model.User{}, &model.Incident{}, &model.Patient{}, &model.Report{}, &model.VitalMeasurement{}); err != nil {
		log.Fatalf("auto migrate: %v", err)
	}

	templateService, err := service.NewTemplateService()
	if err != nil {
		log.Fatalf("load templates: %v", err)
	}

	incidentService := service.NewIncidentService(db)
	eventService := service.NewEventService(db)
	userService := service.NewUserService(db)
	rtr := router.NewRouter()

	rtr.HandleFunc(http.MethodGet, "/users", handler.UserListHandler(userService, templateService))
	rtr.HandleFunc(http.MethodGet, "/user/new", handler.UserFormHandler(userService, templateService))
	rtr.HandleFunc(http.MethodGet, "/users/{id}", handler.UserReadHandler(userService, templateService))
	rtr.HandleFunc(http.MethodPost, "/users", handler.UserCreateHandler(userService, templateService))

	rtr.HandleFunc(http.MethodGet, "/events", handler.EventListHandler(eventService, templateService))
	rtr.HandleFunc(http.MethodGet, "/events/{id}", handler.EventsReadHandler(eventService, templateService))
	rtr.HandleFunc(http.MethodGet, "/event/new", handler.EventFormHandler(eventService, templateService))
	rtr.HandleFunc(http.MethodPost, "/events", handler.EventCreateHandler(eventService, templateService))

	rtr.HandleFunc(http.MethodGet, "/incidents", handler.IncidentListHandler(incidentService, templateService))
	rtr.HandleFunc(http.MethodGet, "/incident/{id}", handler.IncidentReadHandler(incidentService, templateService))
	rtr.HandleFunc(http.MethodGet, "/incidents/new", handler.IncidentFormHandler(eventService, incidentService, templateService))
	rtr.HandleFunc(http.MethodPost, "/incidents", handler.IncidentCreateHandler(eventService, incidentService, templateService))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("listening on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, middleware.SessionMiddleware(rtr)); err != nil {
		log.Fatalf("listen: %v", err)
	}
}
