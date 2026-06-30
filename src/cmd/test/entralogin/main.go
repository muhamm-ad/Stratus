// Command entralogin verifies the single Entra ID sign-in
// end-to-end, WITHOUT the Wails/React frontend.
//
//	STRATUS_ENTRA_TENANT_ID=<tenant> STRATUS_ENTRA_CLIENT_ID=<client> \
//	  go run ./cmd/entralogin

package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/muhamm-ad/stratus/config"
	"github.com/muhamm-ad/stratus/identity/entra"
)

func main() {
	log.SetFlags(0)

	secs, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	cfg, err := entra.ParseConfig(secs.IdentitySection("entra"), os.Getenv)
	if err != nil {
		log.Fatalf("%v\n\nSet STRATUS_ENTRA_TENANT_ID and STRATUS_ENTRA_CLIENT_ID "+
			"(or fill config/config.json) and retry.", err)
	}

	idp := entra.New(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Printf("Tenant : %s", cfg.TenantID)
	log.Println("Opening the system browser to sign in to Microsoft Entra ID...")
	start := time.Now()
	if err := idp.Login(ctx); err != nil {
		log.Fatalf("LOGIN FAILED: %v", err)
	}
	log.Printf("LOGIN OK in %s", time.Since(start).Round(time.Second))
	log.Printf("IsAuthenticated() = %v", idp.IsAuthenticated())
	if _, err := idp.IDToken(ctx); err != nil {
		log.Fatalf("id_token unavailable: %v", err)
	}
	log.Println("id_token acquired — ready for AWS/GCP/Azure exchanges.")
}
