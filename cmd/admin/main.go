package main

import (
	"flag"
	"fmt"
	"litloop/backend/internal/config"
	"litloop/backend/internal/database"
	"litloop/backend/internal/models"
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

func main() {
	email := flag.String("email", "", "Administrator email")
	username := flag.String("username", "admin", "Administrator username")
	flag.Parse()
	if !strings.Contains(*email, "@") {
		log.Fatal("--email is required")
	}
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		fmt.Fprint(os.Stderr, "Password: ")
		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			log.Fatal("Set ADMIN_PASSWORD or use a terminal: ", err)
		}
		fmt.Fprintln(os.Stderr)
		password = string(raw)
	}
	if len(password) < 8 || len(password) > 72 {
		log.Fatal("Password must contain 8–72 bytes")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	db, err := database.Open(config.Load())
	if err != nil {
		log.Fatal(err)
	}
	user := models.User{Email: strings.ToLower(*email), Username: *username, Password: string(hash), IsActive: true, IsStaff: true, IsSuperuser: true}
	if err = db.Create(&user).Error; err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created administrator %d\n", user.ID)
}
