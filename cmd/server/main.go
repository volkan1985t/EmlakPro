package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/volkan1985t/EmlakPro/internal/auth"
	"github.com/volkan1985t/EmlakPro/internal/config"
	"github.com/volkan1985t/EmlakPro/internal/handler"
	"github.com/volkan1985t/EmlakPro/internal/middleware"
	"github.com/volkan1985t/EmlakPro/internal/repository"
	"github.com/volkan1985t/EmlakPro/internal/service"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"golang.org/x/crypto/bcrypt"
)

var version = "dev"

func main() {
	cfgPath := flag.String("config", "config.json", "config dosyası yolu")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("Config yüklenemedi: %v", err)
	}

	db, err := repository.NewDB(cfg.Database)
	if err != nil {
		log.Fatalf("DB bağlantısı kurulamadı: %v", err)
	}
	defer db.Close()
	log.Printf("DB bağlantısı kuruldu: %s:%s/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)

	for _, sub := range []string{"covers", "gallery"} {
		dir := fmt.Sprintf("%s/%s", cfg.App.UploadDir, sub)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Upload dizini oluşturulamadı %s: %v", dir, err)
		}
	}

	userRepo    := repository.NewUserRepository(db)
	listingRepo := repository.NewListingRepository(db)
	requestRepo := repository.NewRequestRepository(db)

	tokenSvc := auth.NewTokenService(
		cfg.Auth.JWTSecret,
		cfg.Auth.AccessTokenTTLMins,
		cfg.Auth.RefreshTokenTTLDays,
	)
	imageSvc := service.NewImageService(cfg)

	if err := ensureAdmin(cfg, userRepo); err != nil {
		log.Fatalf("Admin oluşturulamadı: %v", err)
	}

	authMW := middleware.NewAuthMiddleware(tokenSvc)

	authHandler    := handler.NewAuthHandler(cfg, userRepo, tokenSvc)
	listingHandler := handler.NewListingHandler(cfg, listingRepo, imageSvc)
	uploadHandler  := handler.NewUploadHandler(cfg, imageSvc)
	requestHandler := handler.NewRequestHandler(cfg, requestRepo)
	adminHandler   := handler.NewAdminHandler(cfg, userRepo, listingRepo, requestRepo)
	configHandler  := handler.NewConfigHandler(cfg)

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.App.BaseURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Handle("/uploads/*", http.StripPrefix("/uploads/",
		http.FileServer(http.Dir(cfg.App.UploadDir))))
	r.Handle("/static/*", http.StripPrefix("/static/",
		http.FileServer(http.Dir("./frontend/static"))))

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","version":"%s"}`, version)
	})

	r.Route("/api", func(r chi.Router) {
		r.Post("/auth/login",   authHandler.Login)
		r.Post("/auth/refresh", authHandler.Refresh)
		r.Get("/config",        configHandler.PublicConfig)
		r.Get("/listings/share/{token}", listingHandler.GetByShareToken)

		r.Group(func(r chi.Router) {
			r.Use(authMW.RequireAuth)

			r.Post("/auth/logout", authHandler.Logout)
			r.Get("/auth/me",     authHandler.Me)

			r.Get("/listings",                           listingHandler.List)
			r.Post("/listings",                          listingHandler.Create)
			r.Get("/listings/{id}",                      listingHandler.GetByID)
			r.Put("/listings/{id}",                      listingHandler.Update)
			r.Patch("/listings/{id}/toggle",             listingHandler.ToggleActive)
			r.Delete("/listings/{id}/images/{imgID}",    listingHandler.DeleteImage)

			r.Post("/upload/cover",   uploadHandler.Cover)
			r.Post("/upload/gallery", uploadHandler.Gallery)

			r.Get("/requests",              requestHandler.List)
			r.Post("/requests",             requestHandler.Create)
			r.Put("/requests/{id}",         requestHandler.Update)
			r.Patch("/requests/{id}/toggle", requestHandler.ToggleActive)
			r.Patch("/requests/{id}/notify", requestHandler.ToggleNotify)

			r.Group(func(r chi.Router) {
				r.Use(authMW.RequireAdmin)
				r.Get("/admin/users",               adminHandler.ListUsers)
				r.Post("/admin/users",              adminHandler.CreateUser)
				r.Patch("/admin/users/{id}/toggle", adminHandler.ToggleUser)
				r.Delete("/admin/users/{id}",       adminHandler.DeleteUser)
				r.Get("/admin/listings",            adminHandler.AllListings)
				r.Delete("/admin/listings/{id}",    adminHandler.DeleteListing)
				r.Get("/admin/requests",            adminHandler.AllRequests)
				r.Delete("/admin/requests/{id}",    adminHandler.DeleteRequest)
			})
		})
	})

	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./frontend/templates/index.html")
	})

	srv := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("EmlakPro %s başlatıldı → http://0.0.0.0:%s", version, cfg.App.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Sunucu hatası: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Sunucu kapatılıyor...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("Sunucu durduruldu.")
}

func ensureAdmin(cfg *config.Config, repo *repository.UserRepository) error {
	exists, err := repo.AdminExists()
	if err != nil || exists {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.Admin.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	err = repo.CreateRaw(cfg.Admin.Username, cfg.Admin.Email,
		string(hash), cfg.Admin.FullName, "admin", true)
	if err != nil {
		return fmt.Errorf("admin oluşturulamadı: %w", err)
	}
	log.Printf("Admin kullanıcısı oluşturuldu: %s", cfg.Admin.Username)
	return nil
}
