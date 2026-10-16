package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

// StartServerWithAutocert brings up :80 (only for ACME HTTP-01 challenge,
// without the usual http fallback, as it was before — now there is no nginx before openGo,
// so :80 cannot be occupied by anything else) and :443 with TLS certificate,
// which autocert itself issues and renews via Let's Encrypt.
// The certificate cache is in cacheDir, so as not to reissue them on every restart
// (Let's Encrypt has limits on the number of issuances per week per domain).
func StartServerWithAutocert(handler http.Handler, hosts []string, cacheDir string) {
	certManager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(hosts...),
		Cache:      autocert.DirCache(cacheDir),
	}

	httpServer := &http.Server{
		Addr:    ":80",
		Handler: certManager.HTTPHandler(nil),
	}
	go func() {
		log.Println("> Starting ACME HTTP-01 listener on :80")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Println("ACME HTTP listener error:", err)
		}
	}()

	tlsServer := &http.Server{
		Addr:      ":443",
		Handler:   handler,
		TLSConfig: &tls.Config{GetCertificate: certManager.GetCertificate},
	}
	go func() {
		log.Println("> Starting TLS on :443 (autocert)")
		if err := tlsServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Waiting for Ctrl+C / SIGTERM
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan
	log.Println("> Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := tlsServer.Shutdown(ctx); err != nil {
		log.Println("TLS shutdown error:", err)
	}
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Println("HTTP (ACME) shutdown error:", err)
	}
	log.Println("> Shutdown complete")
}
