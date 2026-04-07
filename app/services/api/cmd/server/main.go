// @title          Treasury Management System API
// @version        1.0
// @description    API for KienlongBank Treasury Management System — Phase 1
// @termsOfService https://kienlongbank.com/terms

// @contact.name  KienlongBank IT Department
// @contact.email it@kienlongbank.com

// @license.name Proprietary
// @license.url  https://kienlongbank.com/license

// @host     localhost:34000
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
// @description                Bearer token (JWT)

// @tag.name        FX
// @tag.description Kinh doanh Ngoại tệ — Foreign Exchange (Spot, Forward, Swap)
// @tag.name        Bonds
// @tag.description Bond Trading — Fixed Income Securities (Government Bond, FI Bond, CD)
// @tag.name        MM
// @tag.description Thị trường Tiền tệ — Money Market
// @tag.name        Limits
// @tag.description Quản lý Hạn mức — Limit Management
// @tag.name        TTQT
// @tag.description Thanh toán Quốc tế — International Payment

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/kienlongbank/treasury-api/internal/admin"
	attachmod "github.com/kienlongbank/treasury-api/internal/attachment"
	auditmod "github.com/kienlongbank/treasury-api/internal/audit"
	"github.com/kienlongbank/treasury-api/internal/auth"
	"github.com/kienlongbank/treasury-api/internal/config"
	"github.com/kienlongbank/treasury-api/internal/database"
	exportmod "github.com/kienlongbank/treasury-api/internal/export"
	bondmod "github.com/kienlongbank/treasury-api/internal/bond"
	dashboardmod "github.com/kienlongbank/treasury-api/internal/dashboard"
	"github.com/kienlongbank/treasury-api/internal/fx"
	notifmod "github.com/kienlongbank/treasury-api/internal/notification"
	creditlimitmod "github.com/kienlongbank/treasury-api/internal/creditlimit"
	"github.com/kienlongbank/treasury-api/internal/limits"
	"github.com/kienlongbank/treasury-api/internal/logger"
	"github.com/kienlongbank/treasury-api/internal/masterdata"
	"github.com/kienlongbank/treasury-api/internal/middleware"
	"github.com/kienlongbank/treasury-api/internal/mm"
	"github.com/kienlongbank/treasury-api/internal/ratelimit"
	"github.com/kienlongbank/treasury-api/internal/repository"
	"github.com/kienlongbank/treasury-api/internal/telemetry"
	settlementmod "github.com/kienlongbank/treasury-api/internal/settlement"
	"github.com/kienlongbank/treasury-api/internal/ttqt"
	"github.com/kienlongbank/treasury-api/pkg/audit"
	"github.com/kienlongbank/treasury-api/pkg/constants"
	"github.com/kienlongbank/treasury-api/pkg/email"
	"github.com/kienlongbank/treasury-api/pkg/export"
	"github.com/kienlongbank/treasury-api/pkg/limitcheck"
	"github.com/kienlongbank/treasury-api/pkg/security"
	"github.com/kienlongbank/treasury-api/pkg/sse"

	_ "github.com/kienlongbank/treasury-api/docs" // swagger docs
)

func main() {
	cfg := config.Load()

	// Initialize Logger
	appLogger, err := logger.New(cfg.Server.Env)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer appLogger.Sync()
	sugar := appLogger.Sugar()

	// Initialize OpenTelemetry
	ctx := context.Background()
	otelShutdown, err := telemetry.InitProvider(ctx, "treasury-api", cfg.Otel.Endpoint)
	if err != nil {
		sugar.Fatalf("Failed to initialize OpenTelemetry: %v", err)
	}
	defer func() {
		if err := otelShutdown(ctx); err != nil {
			sugar.Errorf("Error during OpenTelemetry shutdown: %v", err)
		}
	}()

	// Database Connection
	var pool *pgxpool.Pool
	if cfg.DB.URL != "" {
		p, err := database.Connect(ctx, cfg.DB)
		if err != nil {
			sugar.Warnf("Failed to connect to database: %v - running in mock mode", err)
		} else {
			pool = p
			defer pool.Close()
		}
	} else {
		sugar.Warn("DATABASE_URL not configured - running in mock mode")
	}

	// Sync permissions from Go constants → DB (single source of truth)
	if pool != nil {
		syncPermissions(ctx, pool, sugar)
	}

	// Dependencies
	jwtCfg := security.LoadJWTConfig()
	jwtMgr := security.NewJWTManager(jwtCfg)
	rbacChecker := security.NewRBACChecker()

	// Security config
	securityCfg := config.LoadSecurityConfig()

	// Redis Rate Limiter (optional — if Redis is unavailable, rate limiting is disabled)
	var rateLimiter *ratelimit.Limiter
	if securityCfg.RedisURL != "" {
		rl, err := ratelimit.New(securityCfg.RedisURL)
		if err != nil {
			sugar.Warnf("⚠️ Redis unavailable: %v — rate limiting disabled", err)
		} else {
			rateLimiter = rl
			sugar.Info("Redis rate limiter connected")
		}
	}
	defer func() {
		if rateLimiter != nil {
			_ = rateLimiter.Close()
		}
	}()

	// Auth Module
	authRepo := auth.NewRepository(pool)
	authService := auth.NewService(authRepo, jwtMgr, rbacChecker, &securityCfg, appLogger)
	authHandler := auth.NewHandler(authService, &securityCfg, appLogger)

	// Audit Logger (shared)
	auditLogger := audit.NewLogger(pool, appLogger)

	// Credit Limit + FX Module — wired with repository + service + handler + audit
	limitRepo := limits.NewRepository(pool)
	fxRepo := fx.NewRepository(pool)
	fxService := fx.NewService(fxRepo, authRepo, rbacChecker, auditLogger, pool, appLogger)
	if pool != nil {
		limitChecker := limitcheck.NewChecker(limitRepo, fxRepo, appLogger)
		fxService.SetLimitChecker(limitChecker)
	}
	fxHandler := fx.NewHandler(fxService, appLogger)

	// Bond Module (Fixed Income Securities)
	bondRepo := bondmod.NewRepository(pool)
	bondService := bondmod.NewService(bondRepo, authRepo, rbacChecker, auditLogger, pool, appLogger)
	bondHandler := bondmod.NewHandler(bondService, appLogger)

	// SSE Broker + Notification Module
	sseBroker := sse.NewBroker(appLogger)
	var notifHandler *notifmod.Handler
	var notifService *notifmod.Service
	if pool != nil {
		notifRepo := repository.NewNotificationRepo(pool)
		notifService = notifmod.NewService(notifRepo, sseBroker, appLogger)
		notifHandler = notifmod.NewHandler(notifService, sseBroker, appLogger)
		fxService.SetNotifier(notifService)
		bondService.SetNotifier(notifService)
	}

	// Email Service (optional — requires database)
	var emailService *email.Service
	var emailWorker *email.Worker
	var emailHealthHandler *admin.EmailHealthHandler
	if pool != nil {
		emailTemplates, err := email.NewTemplateRenderer()
		if err != nil {
			sugar.Fatalf("Failed to load email templates: %v", err)
		}
		smtpSender := email.NewSMTPSender(cfg.Email, appLogger)
		outboxRepo := email.NewPgOutboxRepository(pool)
		emailWorker = email.NewWorker(outboxRepo, smtpSender, emailTemplates, appLogger, cfg.Email.RateLimit, cfg.Email.BurstSize)
		emailService = email.NewService(outboxRepo, emailWorker, emailTemplates, cfg.Email, appLogger)
		emailWorker.Start(ctx)
		fxService.SetEmailer(emailService)
		bondService.SetEmailer(emailService)
		emailHealthHandler = admin.NewEmailHealthHandler(outboxRepo, appLogger)
		sugar.Info("Email service started")
	}

	// Admin Module
	adminRepo := admin.NewRepository(pool)
	adminService := admin.NewService(adminRepo, rbacChecker, auditLogger, appLogger)
	adminHandler := admin.NewHandler(adminService, appLogger)

	// Master Data Module
	cpRepo := masterdata.NewCounterpartyRepository(pool)
	mdRepo := masterdata.NewMasterDataRepository(pool)
	mdHandler := masterdata.NewHandler(cpRepo, mdRepo, auditLogger, appLogger)

	// Audit Module
	auditRepo := auditmod.NewRepository(pool)
	auditHandler := auditmod.NewHandler(auditRepo, appLogger)

	// Export Engine
	exportAuditRepo := repository.NewExportAuditRepo(pool)
	exportCfg := export.ExportConfig{
		RetentionDays:  90,
		MinioBucket:    os.Getenv("MINIO_BUCKET"),
		MinioEndpoint:  os.Getenv("MINIO_ENDPOINT"),
		MinioAccessKey: os.Getenv("MINIO_ACCESS_KEY"),
		MinioSecretKey: os.Getenv("MINIO_SECRET_KEY"),
		MinioUseSSL:    os.Getenv("MINIO_USE_SSL") == "true",
	}
	if exportCfg.MinioBucket == "" {
		exportCfg.MinioBucket = "treasury-exports"
	}
	exportEngine, err := export.NewEngine(exportAuditRepo, exportCfg, appLogger)
	if err != nil {
		sugar.Warnf("Failed to create export engine: %v — export disabled", err)
	}
	fxExportHandler := fx.NewExportHandler(fxService, exportEngine, appLogger)
	bondExportHandler := bondmod.NewExportHandler(bondService, exportEngine, appLogger)
	exportHistoryHandler := exportmod.NewHandler(exportEngine, appLogger)

	// Attachment Module (shared for all deal modules)
	var attachHandler *attachmod.Handler
	attachRepo := repository.NewAttachmentRepo(pool)
	if pool != nil {
		fxService.SetAttachmentRepo(attachRepo)
		// Create a dedicated MinIO client for attachments (same credentials, different bucket)
		if exportCfg.MinioEndpoint != "" {
			attachMinioClient, err := minio.New(exportCfg.MinioEndpoint, &minio.Options{
				Creds:  credentials.NewStaticV4(exportCfg.MinioAccessKey, exportCfg.MinioSecretKey, ""),
				Secure: exportCfg.MinioUseSSL,
			})
			if err != nil {
				sugar.Warnf("Failed to create attachment MinIO client: %v — attachments disabled", err)
			} else {
				attachHandler = attachmod.NewHandler(attachRepo, attachMinioClient, appLogger)
				sugar.Info("Attachment handler ready")
			}
		}
	}

	// Credit Limit Module (BRD §3.4)
	creditLimitRepo := creditlimitmod.NewRepository(pool)
	creditLimitService := creditlimitmod.NewService(creditLimitRepo, authRepo, cpRepo, rbacChecker, auditLogger, appLogger)
	creditLimitHandler := creditlimitmod.NewHandler(creditLimitService, appLogger)

	// MM — Money Market Module (Interbank + OMO + Government Repo)
	mmInterbankRepo := mm.NewInterbankRepository(pool)
	mmInterbankService := mm.NewInterbankService(mmInterbankRepo, authRepo, rbacChecker, auditLogger, pool, appLogger)
	mmOMORepoRepo := mm.NewOMORepoRepository(pool)
	mmOMORepoService := mm.NewOMORepoService(mmOMORepoRepo, authRepo, rbacChecker, auditLogger, pool, appLogger)
	if notifService != nil {
		mmInterbankService.SetNotifier(notifService)
		mmOMORepoService.SetNotifier(notifService)
	}
	if emailService != nil {
		mmInterbankService.SetEmailer(emailService)
		mmOMORepoService.SetEmailer(emailService)
	}
	mmHandler := mm.NewHandler(mmInterbankService, mmOMORepoService, appLogger)
	mmExportHandler := mm.NewExportHandler(mmInterbankService, exportEngine, appLogger)

	_ = limits.NewHandler(pool) // legacy placeholder — replaced by creditlimitmod
	_ = ttqt.NewHandler(pool) // legacy placeholder — replaced by settlementmod
	settlementRepo := settlementmod.NewRepository(pool)
	settlementService := settlementmod.NewService(settlementRepo, authRepo, rbacChecker, auditLogger, appLogger)
	settlementHandler := settlementmod.NewHandler(settlementService, appLogger)

	// Dashboard — aggregate views (no permission check, read-only aggregate data)
	dashboardRepo := dashboardmod.NewRepository(pool)
	dashboardService := dashboardmod.NewService(dashboardRepo)
	dashboardHandler := dashboardmod.NewHandler(dashboardService, appLogger)

	// Router and Middleware
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(telemetry.Middleware("treasury-api"))
	r.Use(logger.Middleware(appLogger))
	r.Use(middleware.Recovery)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           cfg.CORS.MaxAge,
	}))

	// Public routes
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"treasury-api"}`))
	})

	// Swagger UI
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// API v1 — Auth (public routes, no auth middleware)
	r.Route("/api/v1/auth", func(r chi.Router) {
		if rateLimiter != nil {
			r.With(middleware.LoginRateLimit(rateLimiter, securityCfg.LoginRateLimit, securityCfg.LoginRateWindow)).
				Post("/login", authHandler.Login)
			r.With(middleware.RateLimit(rateLimiter, securityCfg.RefreshRateLimit, securityCfg.RefreshRateWindow, func(req *http.Request) string {
				return ratelimit.IPKey(req.RemoteAddr)
			})).Post("/refresh", authHandler.Refresh)
		} else {
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
		}
		r.Post("/logout", authHandler.Logout)

		// Auth — protected routes (require auth middleware)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(jwtMgr))
			r.Get("/me", authHandler.Me)
			r.Post("/password", authHandler.ChangePassword)
			r.Post("/logout-all", authHandler.LogoutAll)
			r.Get("/sessions", authHandler.ListSessions)
			r.Delete("/sessions/{id}", authHandler.RevokeSession)
		})
	})

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Use auth middleware for all v1 routes
		r.Use(middleware.Auth(jwtMgr))

		// Global API rate limiting for authenticated routes
		if rateLimiter != nil {
			r.Use(middleware.APIRateLimit(rateLimiter, securityCfg.APIRateLimit, securityCfg.APIRateWindow))
		}

		// FX — Kinh doanh Ngoại tệ
		r.Route("/fx", func(r chi.Router) {
			r.With(middleware.RequirePermission(rbacChecker, constants.PermFxView)).Get("/", fxHandler.List)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermFxCreate)).Post("/", fxHandler.Create)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermFxView)).Get("/{id}", fxHandler.Get)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermFxEdit)).Put("/{id}", fxHandler.Update)
			// Approve: multiple roles can approve — router checks broad "has any approve permission", service checks specific level
			r.With(middleware.RequireAnyPermission(rbacChecker,
				constants.PermFxApproveL1, constants.PermFxApproveL2,
				constants.PermFxBookL1, constants.PermFxBookL2,
				constants.PermFxSettle,
			)).Post("/{id}/approve", fxHandler.Approve)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermFxRecall)).Post("/{id}/recall", fxHandler.Recall)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermFxCancelRequest)).Post("/{id}/cancel", fxHandler.Cancel)
			r.With(middleware.RequireAnyPermission(rbacChecker,
				constants.PermFxCancelApproveL1, constants.PermFxCancelApproveL2,
			)).Post("/{id}/cancel-approve", fxHandler.CancelApprove)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermFxView)).Get("/{id}/history", fxHandler.History)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermFxClone)).Post("/{id}/clone", fxHandler.Clone)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermFxDelete)).Delete("/{id}", fxHandler.Delete)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermFxExport)).Post("/deals/export", fxExportHandler.ExportDeals)
		})

		// Export History
		r.Route("/exports", func(r chi.Router) {
			r.Get("/", exportHistoryHandler.List)
			r.Get("/{code}", exportHistoryHandler.Get)
			r.Get("/{code}/download", exportHistoryHandler.Download)
		})

		// Attachments — any authenticated user can upload/download
		if attachHandler != nil {
			r.Route("/attachments", func(r chi.Router) {
				r.Post("/upload", attachHandler.Upload)
				r.Get("/{id}/download", attachHandler.Download)
				r.Get("/deal/{module}/{dealId}", attachHandler.ListByDeal)
				r.Delete("/{id}", attachHandler.Delete)
			})
		}

		// Notifications — any authenticated user
		if notifHandler != nil {
			r.Route("/notifications", func(r chi.Router) {
				r.Get("/", notifHandler.List)
				r.Get("/stream", notifHandler.Stream)
				r.Get("/unread-count", notifHandler.UnreadCount)
				r.Post("/{id}/read", notifHandler.MarkRead)
				r.Post("/read-all", notifHandler.MarkAllRead)
			})
		}

		// Counterparties — public list for deal creation (any authenticated user)
		r.Route("/counterparties", func(r chi.Router) {
			r.Get("/", mdHandler.ListCounterparties)
			r.Get("/{id}", mdHandler.GetCounterparty)
			// Admin CRUD (requires MASTER_DATA.MANAGE)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermMasterDataManage)).Post("/", mdHandler.CreateCounterparty)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermMasterDataManage)).Put("/{id}", mdHandler.UpdateCounterparty)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermMasterDataManage)).Delete("/{id}", mdHandler.DeleteCounterparty)
		})

		// Currencies — any authenticated user
		r.Get("/currencies", mdHandler.ListCurrencies)

		// Currency Pairs — any authenticated user
		r.Get("/currency-pairs", mdHandler.ListCurrencyPairs)

		// Branches — any authenticated user
		r.Get("/branches", mdHandler.ListBranches)

		// Exchange Rates — any authenticated user
		r.Route("/exchange-rates", func(r chi.Router) {
			r.Get("/", mdHandler.ListExchangeRates)
			r.Get("/latest", mdHandler.GetLatestRate)
		})

		// Admin — User Management (requires SYSTEM.MANAGE)
		r.Route("/admin", func(r chi.Router) {
			r.Route("/users", func(r chi.Router) {
				r.Use(middleware.RequirePermission(rbacChecker, constants.PermSystemManage))
				r.Get("/", adminHandler.ListUsers)
				r.Post("/", adminHandler.CreateUser)
				r.Get("/{id}", adminHandler.GetUser)
				r.Put("/{id}", adminHandler.UpdateUser)
				r.Post("/{id}/lock", adminHandler.LockUser)
				r.Post("/{id}/unlock", adminHandler.UnlockUser)
				r.Post("/{id}/reset-password", adminHandler.ResetPassword)
				r.Post("/{id}/roles", adminHandler.AssignRole)
				r.Delete("/{id}/roles/{code}", adminHandler.RevokeRole)
			})

			r.Route("/roles", func(r chi.Router) {
				r.Use(middleware.RequirePermission(rbacChecker, constants.PermSystemManage))
				r.Get("/", adminHandler.ListRoles)
				r.Get("/{code}/permissions", adminHandler.GetRolePermissions)
				r.Put("/{code}/permissions", adminHandler.UpdateRolePermissions)
			})

			r.With(middleware.RequirePermission(rbacChecker, constants.PermSystemManage)).
				Get("/permissions", adminHandler.ListPermissions)

			r.Route("/audit-logs", func(r chi.Router) {
				r.Use(middleware.RequirePermission(rbacChecker, constants.PermAuditLogView))
				r.Get("/", auditHandler.List)
				r.Get("/stats", auditHandler.Stats)
			})

			// Email Health (requires SYSTEM.MANAGE)
			if emailHealthHandler != nil {
				r.Route("/email-health", func(r chi.Router) {
					r.Use(middleware.RequirePermission(rbacChecker, constants.PermSystemManage))
					r.Get("/", emailHealthHandler.Health)
				})
				r.Route("/email-outbox", func(r chi.Router) {
					r.Use(middleware.RequirePermission(rbacChecker, constants.PermSystemManage))
					r.Get("/", emailHealthHandler.ListOutbox)
				})
			}
		})

		// Bonds — Fixed Income Securities
		r.Route("/bonds", func(r chi.Router) {
			r.With(middleware.RequirePermission(rbacChecker, constants.PermBondView)).Get("/", bondHandler.List)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermBondCreate)).Post("/", bondHandler.Create)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermBondView)).Get("/inventory", bondHandler.Inventory)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermBondView)).Get("/{id}", bondHandler.Get)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermBondEdit)).Put("/{id}", bondHandler.Update)
			r.With(middleware.RequireAnyPermission(rbacChecker,
				constants.PermBondApproveL1, constants.PermBondApproveL2,
				constants.PermBondBookL1, constants.PermBondBookL2,
			)).Post("/{id}/approve", bondHandler.Approve)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermBondRecall)).Post("/{id}/recall", bondHandler.Recall)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermBondCancelRequest)).Post("/{id}/cancel", bondHandler.Cancel)
			r.With(middleware.RequireAnyPermission(rbacChecker,
				constants.PermBondCancelApproveL1, constants.PermBondCancelApproveL2,
			)).Post("/{id}/cancel-approve", bondHandler.CancelApprove)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermBondView)).Get("/{id}/history", bondHandler.History)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermBondClone)).Post("/{id}/clone", bondHandler.Clone)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermBondDelete)).Delete("/{id}", bondHandler.Delete)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermBondExport)).Post("/deals/export", bondExportHandler.ExportDeals)
		})

		// MM — Thị trường Tiền tệ (Interbank + OMO + Government Repo)
		r.Route("/mm", func(r chi.Router) {
			// Interbank
			r.Route("/interbank", func(r chi.Router) {
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMInterbankView)).Get("/", mmHandler.InterbankList)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMInterbankCreate)).Post("/", mmHandler.InterbankCreate)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMInterbankView)).Get("/{id}", mmHandler.InterbankGet)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMInterbankEdit)).Put("/{id}", mmHandler.InterbankUpdate)
				r.With(middleware.RequireAnyPermission(rbacChecker,
					constants.PermMMInterbankApproveL1, constants.PermMMInterbankApproveL2,
					constants.PermMMInterbankApproveRiskL1, constants.PermMMInterbankApproveRiskL2,
					constants.PermMMInterbankBookL1, constants.PermMMInterbankBookL2,
					constants.PermMMInterbankSettle,
				)).Post("/{id}/approve", mmHandler.InterbankApprove)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMInterbankRecall)).Post("/{id}/recall", mmHandler.InterbankRecall)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMInterbankCancelRequest)).Post("/{id}/cancel", mmHandler.InterbankCancel)
				r.With(middleware.RequireAnyPermission(rbacChecker,
					constants.PermMMInterbankCancelApproveL1, constants.PermMMInterbankCancelApproveL2,
				)).Post("/{id}/cancel-approve", mmHandler.InterbankCancelApprove)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMInterbankView)).Get("/{id}/history", mmHandler.InterbankHistory)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMInterbankClone)).Post("/{id}/clone", mmHandler.InterbankClone)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMInterbankDelete)).Delete("/{id}", mmHandler.InterbankDelete)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMInterbankExport)).Post("/export", mmExportHandler.ExportDeals)
			})

			// OMO
			r.Route("/omo", func(r chi.Router) {
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMOMORepoView)).Get("/", mmHandler.OMOList)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMOMORepoCreate)).Post("/", mmHandler.OMOCreate)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMOMORepoView)).Get("/{id}", mmHandler.OMOGet)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMOMORepoEdit)).Put("/{id}", mmHandler.OMOUpdate)
				r.With(middleware.RequireAnyPermission(rbacChecker,
					constants.PermMMOMORepoApproveL1, constants.PermMMOMORepoApproveL2,
					constants.PermMMOMORepoBookL1, constants.PermMMOMORepoBookL2,
				)).Post("/{id}/approve", mmHandler.OMOApprove)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMOMORepoRecall)).Post("/{id}/recall", mmHandler.OMORecall)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMOMORepoCancelRequest)).Post("/{id}/cancel", mmHandler.OMOCancel)
				r.With(middleware.RequireAnyPermission(rbacChecker,
					constants.PermMMOMORepoCancelApproveL1, constants.PermMMOMORepoCancelApproveL2,
				)).Post("/{id}/cancel-approve", mmHandler.OMOCancelApprove)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMOMORepoView)).Get("/{id}/history", mmHandler.OMOHistory)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMOMORepoClone)).Post("/{id}/clone", mmHandler.OMOClone)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMOMORepoDelete)).Delete("/{id}", mmHandler.OMODelete)
			})

			// Government Repo
			r.Route("/govt-repo", func(r chi.Router) {
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMOMORepoView)).Get("/", mmHandler.RepoList)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMOMORepoCreate)).Post("/", mmHandler.RepoCreate)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMOMORepoView)).Get("/{id}", mmHandler.RepoGet)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMOMORepoEdit)).Put("/{id}", mmHandler.RepoUpdate)
				r.With(middleware.RequireAnyPermission(rbacChecker,
					constants.PermMMOMORepoApproveL1, constants.PermMMOMORepoApproveL2,
					constants.PermMMOMORepoBookL1, constants.PermMMOMORepoBookL2,
				)).Post("/{id}/approve", mmHandler.RepoApprove)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMOMORepoRecall)).Post("/{id}/recall", mmHandler.RepoRecall)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMOMORepoCancelRequest)).Post("/{id}/cancel", mmHandler.RepoCancel)
				r.With(middleware.RequireAnyPermission(rbacChecker,
					constants.PermMMOMORepoCancelApproveL1, constants.PermMMOMORepoCancelApproveL2,
				)).Post("/{id}/cancel-approve", mmHandler.RepoCancelApprove)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMOMORepoView)).Get("/{id}/history", mmHandler.RepoHistory)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMOMORepoClone)).Post("/{id}/clone", mmHandler.RepoClone)
				r.With(middleware.RequirePermission(rbacChecker, constants.PermMMOMORepoDelete)).Delete("/{id}", mmHandler.RepoDelete)
			})
		})

		// Limits — Quản lý Hạn mức (BRD §3.4)
		r.Route("/limits", func(r chi.Router) {
			r.With(middleware.RequirePermission(rbacChecker, constants.PermCreditLimitView)).
				Get("/", creditLimitHandler.List)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermCreditLimitView)).
				Get("/daily-summary", creditLimitHandler.DailySummary)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermCreditLimitView)).
				Post("/daily-summary/export", creditLimitHandler.ExportDailySummary)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermCreditLimitView)).
				Get("/approvals", creditLimitHandler.ListApprovals)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermCreditLimitApproveRiskL1)).
				Post("/approve", creditLimitHandler.ApproveRiskOfficer)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermCreditLimitApproveRiskL1)).
				Post("/reject", creditLimitHandler.RejectRiskOfficer)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermCreditLimitApproveRiskL2)).
				Post("/approve-head", creditLimitHandler.ApproveRiskHead)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermCreditLimitApproveRiskL2)).
				Post("/reject-head", creditLimitHandler.RejectRiskHead)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermCreditLimitView)).
				Get("/utilization/{counterpartyId}", creditLimitHandler.GetUtilization)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermCreditLimitView)).
				Get("/{counterpartyId}", creditLimitHandler.GetByCounterparty)
			r.With(middleware.RequireAnyPermission(rbacChecker,
				constants.PermCreditLimitCreate, constants.PermCreditLimitApproveL1)).
				Put("/{counterpartyId}", creditLimitHandler.SetLimit)
		})

		// TTQT — Thanh toán Quốc tế
		// Settlements — International Payments (TTQT)
		r.Route("/settlements", func(r chi.Router) {
			r.With(middleware.RequirePermission(rbacChecker, constants.PermIntlPaymentView)).
				Get("/", settlementHandler.List)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermIntlPaymentView)).
				Get("/{id}", settlementHandler.Get)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermIntlPaymentSettle)).
				Post("/{id}/approve", settlementHandler.Approve)
			r.With(middleware.RequirePermission(rbacChecker, constants.PermIntlPaymentSettle)).
				Post("/{id}/reject", settlementHandler.Reject)
		})

		// Dashboard — aggregate overview (any authenticated user)
		r.Get("/dashboard", dashboardHandler.Get)
	})

	// Server with graceful shutdown
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:           r,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		sugar.Infof("Treasury API server starting at :%s (%s)", cfg.Server.Port, cfg.Server.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			sugar.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	sugar.Info("Shutting down server...")

	// Stop email worker before closing DB
	if emailWorker != nil {
		sugar.Info("Stopping email worker...")
		emailWorker.Stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		sugar.Fatalf("Server forced to shutdown: %v", err)
	}

	sugar.Info("Server stopped")
}

// syncPermissions upserts all Go-defined permissions into DB and syncs role_permissions.
// This ensures DB always matches Go constants — add new permissions in constants/permissions.go
// and they auto-appear in DB on next startup.
func syncPermissions(ctx context.Context, pool *pgxpool.Pool, sugar interface {
	Infof(template string, args ...interface{})
	Warnf(template string, args ...interface{})
}) {
	defs := constants.AllPermissionDefs()

	tx, err := pool.Begin(ctx)
	if err != nil {
		sugar.Warnf("Permission sync: failed to begin tx: %v", err)
		return
	}
	defer tx.Rollback(ctx)

	// 1. Upsert permissions
	upserted := 0
	for _, d := range defs {
		tag, err := tx.Exec(ctx, `
			INSERT INTO permissions (code, resource, action, description)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (code) DO UPDATE SET resource = $2, action = $3`,
			d.Code, d.Resource, d.Action, d.Description)
		if err != nil {
			sugar.Warnf("Permission sync: failed to upsert %s: %v", d.Code, err)
			return
		}
		if tag.RowsAffected() > 0 {
			upserted++
		}
	}

	// 2. Sync role_permissions from constants (only if DB has no custom overrides)
	for roleCode, permCodes := range constants.RolePermissions {
		// Check if role has any permissions in DB already
		var count int
		err := tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM role_permissions rp
			JOIN roles r ON r.id = rp.role_id
			WHERE r.code = $1`, roleCode).Scan(&count)
		if err != nil || count > 0 {
			// Role already has permissions in DB — don't override (admin may have customized)
			continue
		}

		// No permissions in DB → seed from constants
		for _, code := range permCodes {
			_, err := tx.Exec(ctx, `
				INSERT INTO role_permissions (role_id, permission_id)
				SELECT r.id, p.id FROM roles r, permissions p
				WHERE r.code = $1 AND p.code = $2
				ON CONFLICT DO NOTHING`,
				roleCode, code)
			if err != nil {
				sugar.Warnf("Permission sync: failed to seed %s → %s: %v", roleCode, code, err)
			}
		}
		sugar.Infof("Permission sync: seeded %d permissions for role %s", len(permCodes), roleCode)
	}

	if err := tx.Commit(ctx); err != nil {
		sugar.Warnf("Permission sync: failed to commit: %v", err)
		return
	}

	sugar.Infof("Permission sync: %d definitions, %d upserted", len(defs), upserted)
}
