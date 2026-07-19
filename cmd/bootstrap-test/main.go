package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asaidimu/hestia/internal/boot"
	"github.com/asaidimu/hestia/internal/app"
	"github.com/asaidimu/hestia/internal/interface/api"
	"github.com/asaidimu/hestia/migrations"
)

func buildInterface(a *boot.Application, mod *app.SystemModule) *api.Interface {
	secureDisp := mod.SecureDispatcher(a.Dispatcher())
	return api.New(api.Options{
		Dispatcher:          secureDisp,
		InternalDispatcher:  a.Dispatcher(),
		CredentialsProvider: mod.CredentialsProvider(),
		Logger:              a.Loggers.File,
		Addr:                a.Config.Port,
		Registrations:       a.Registrations,
		CookieConfig:        a.Config.CookieConfig,
		SessionTTL:          a.Config.SessionTTL,
		IdleTTL:             a.Config.IdleTTL,
		RefreshTTL:          a.Config.RefreshTTL,
		APIPrefix:           a.Config.APIPrefix,
		UserModel:           mod.UserModel(),
	})
}

func main() {
	cfg, err := boot.NewConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}
	application := boot.Create(cfg)
	defer application.Close()

	if err := migrations.Apply(context.Background(), application.Persistence()); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: migrations: %v\n", err)
		os.Exit(1)
	}

	systemMod := app.New(cfg, application.Dispatcher(), app.Options{
		Logger:            application.Loggers.File,
		AdminEmail:        cfg.AdminEmail,
		AdminPassword:     cfg.AdminPassword,
		ForceBootstrapped: cfg.ForceBootstrapped,
		OnBootstrapped: func() {
			fmt.Println("=== OnBootstrapped CALLED ===")
			application.RestartAll(true)
		},
		OnReset: func() {
			fmt.Println("=== OnReset CALLED ===")
			application.Reset(cfg, "dev")
		},
	})

	if err := application.RegisterModules(systemMod); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: %v\n", err)
		os.Exit(1)
	}

	// Always seed default policies so the test server has a complete set
	// of operations and rules regardless of bootstrapped state.
	ctx := context.Background()
	if err := systemMod.SeedPolicies(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: seed policies: %v\n", err)
		os.Exit(1)
	}

	ephemeralKey := systemMod.EphemeralKey()
	adminEmail := systemMod.AdminEmail()
	bootstrapped := systemMod.Bootstrapped()

	fmt.Printf("Bootstrapped: %v\n", bootstrapped)
	fmt.Printf("Admin email:  %s\n", adminEmail)
	if !bootstrapped && ephemeralKey != "" {
		fmt.Printf("Ephemeral key: %s\n", ephemeralKey)
	}

	iface := buildInterface(application, systemMod)
	application.AddInterface(iface)
	application.Start(bootstrapped)

	if bootstrapped {
		fmt.Println("\n=== System already bootstrapped, skipping bootstrap flow ===")
	} else {
		fmt.Println("\n=== Running bootstrap flow ===")

		port := cfg.Port
		url := fmt.Sprintf("http://localhost%s/api/bootstrap/password", port)

		var healthErr error
		client := &http.Client{Timeout: 10 * time.Second}
		for i := 0; i < 10; i++ {
			var hresp *http.Response
			hresp, healthErr = client.Get(fmt.Sprintf("http://localhost%s/api/health", port))
			if healthErr == nil {
				hresp.Body.Close()
				break
			}
			fmt.Printf("  Waiting for server... attempt %d\n", i+1)
			time.Sleep(500 * time.Millisecond)
		}
		if healthErr != nil {
			fmt.Fprintf(os.Stderr, "Server did not start: %v\n", healthErr)
			os.Exit(1)
		}
		fmt.Println("  Server is ready")

		body, _ := json.Marshal(map[string]string{
			"password": "newpassword123",
			"email":    "admin@example.com",
		})

		req, _ := http.NewRequest("PUT", url, bytes.NewReader(body))
		req.Header.Set("X-API-Key", ephemeralKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Bootstrap request failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		fmt.Printf("Bootstrap response: %s\n", string(respBody))
		fmt.Printf("Bootstrap status:  %d\n", resp.StatusCode)

		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "Bootstrap failed with status %d\n", resp.StatusCode)
			os.Exit(1)
		}

		fmt.Println("\n=== Waiting for restart (5s) ===")
		time.Sleep(5 * time.Second)

		fmt.Println("\n=== Checking if system is now bootstrapped ===")
		healthURL := fmt.Sprintf("http://localhost%s/api/health", port)
		healthResp, err := client.Get(healthURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Health check failed: %v\n", err)
			os.Exit(1)
		}
		defer healthResp.Body.Close()
		healthBody, _ := io.ReadAll(healthResp.Body)
		fmt.Printf("Health response: %s\n", string(healthBody))

		var healthResult struct {
			Bootstrapped bool `json:"bootstrapped"`
		}
		json.Unmarshal(healthBody, &healthResult)

		if healthResult.Bootstrapped {
			fmt.Println("\n=== SUCCESS: System is bootstrapped after bootstrap flow ===")
		} else {
			fmt.Fprintf(os.Stderr, "\n=== FAILURE: System is NOT bootstrapped after bootstrap flow ===\n")
			os.Exit(1)
		}
	}

	fmt.Println("\n=== Test complete, shutting down ===")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
	case <-time.After(2 * time.Second):
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	application.Shutdown(ctx)
	fmt.Println("=== Done ===")
}
