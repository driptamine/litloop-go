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
		raw, e := term.ReadPassword(int(os.Stdin.Fd()))
		if e != nil {
			log.Fatal("Set ADMIN_PASSWORD or use a terminal: ", e)
		}
		fmt.Fprintln(os.Stderr)
		password = string(raw)
	}
	if len(password) < 8 || len(password) > 72 {
		log.Fatal("Password must contain 8–72 bytes")
	}
	hash, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if e != nil {
		log.Fatal(e)
	}
	db, e := database.Open(config.Load())
	if e != nil {
		log.Fatal(e)
	}
	u := models.User{Email: strings.ToLower(*email), Username: *username, Password: string(hash), IsActive: true, IsStaff: true, IsSuperuser: true}
	if e = db.Create(&u).Error; e != nil {
		log.Fatal(e)
	}
	fmt.Printf("Created administrator %d\n", u.ID)
}
