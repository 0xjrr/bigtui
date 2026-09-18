package auth

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"golang.org/x/oauth2/google"
)

const cloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

func EnsureApplicationDefaultCredentials(ctx context.Context) error {
	if credentialsAvailable(ctx) {
		return nil
	}

	fmt.Println("No Google Cloud credentials found. Starting gcloud authentication...")
	command := exec.CommandContext(ctx, "gcloud", "auth", "application-default", "login")
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("gcloud authentication failed: %w", err)
	}
	return nil
}

func credentialsAvailable(ctx context.Context) bool {
	credentials, err := google.FindDefaultCredentials(ctx, cloudPlatformScope)
	if err != nil || credentials == nil || credentials.TokenSource == nil {
		return false
	}
	_, err = credentials.TokenSource.Token()
	return err == nil
}
