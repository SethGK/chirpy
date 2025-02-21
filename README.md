# Chirpy

Chirpy is a micro-blogging REST API built with Go. Users can post short messages (chirps), manage their profiles, and upgrade to premium "Chirpy Red" membership via webhook events. The API uses JWTs for access control and refresh tokens for session management.

## What Does Chirpy Do?

- **User Management:**  
  Register, log in, update your profile (email & password), and delete your chirps.
- **Chirp Functionality:**  
  Create, retrieve, update, and delete chirps. Only the author of a chirp can delete it.

- **Premium Membership:**  
  Receive webhook notifications from Polka to upgrade users to Chirpy Red, granting extra features.
- **Secure Authentication:**  
  Uses JWTs that expire after 1 hour for access and refresh tokens that expire after 60 days.

## Installation and Setup

1. **Clone the Repository:**
   ```bash
   git clone https://github.com/SethGK/chirpy.git
   cd chirpy

2. **Create a .env file in the project root with:**
    ```bash
    DB_URL=postgres://user:password@localhost:5432/chirpy?sslmode=disable
    JWT_SECRET=your_jwt_secret_here
    POLKA_KEY=f271c81ff7084ee5b99a5091b42d486e
    PLATFORM=dev

3. **Run the database migration and generate sqlc code**
    ```bash
    goose Up
    sqlc generate

4. **Run the application**
    ```bash
    go run .

