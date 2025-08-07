# Bookmark Manager API

> **Note:** This project was created for learning purposes to practice Go. It is not intended for production use.

A RESTful API for managing bookmarks with user authentication, built primarily with the Go standard library.

## Features

*   **User Authentication:** Secure user registration and login using JWT tokens.
*   **Bookmark Management:** Full CRUD (Create, Read, Update, Delete) operations for bookmarks.
*   **Tagging System:** Organize bookmarks with tags.
*   **Powerful Search:** Filter bookmarks by tags or search through titles, descriptions, and notes.
*   **Pagination:** Efficiently browse through large collections of bookmarks.
*   **SQLite Backend:** Uses a lightweight and file-based SQLite database for storage.

## Getting Started

### Prerequisites

*   Go (version 1.20 or higher recommended)
*   SQLite

### Installation & Running

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/your-username/bookmarks-manager-api.git
    cd bookmarks-manager-api
    ```

2.  **Install dependencies:**
    ```bash
    go mod tidy
    ```

3.  **Run the server:**
    ```bash
    go run cmd/serve.go
    ```
    The API will be available at `http://localhost:3000`.

## API Endpoints

A brief overview of the available endpoints. For detailed information, please refer to the [OpenAPI Specification](api.yaml) or the [Project Specifications](SPECS.md).

### Authentication

*   `POST /api/auth/register`: Register a new user.
*   `POST /api/auth/login`: Log in and receive a JWT token.
*   `GET /api/auth/me`: Get the current user's profile.

### Bookmarks

*   `GET /api/bookmarks`: List all bookmarks with filtering and pagination.
*   `POST /api/bookmarks`: Create a new bookmark.
*   `GET /api/bookmarks/{id}`: Get a single bookmark by its ID.
*   `PUT /api/bookmarks/{id}`: Update a bookmark.
*   `DELETE /api/bookmarks/{id}`: Delete a bookmark.

### Tags

*   `GET /api/tags`: List all tags for the current user.
*   `DELETE /api/tags/{id}`: Delete a tag.

## Usage Example

Here's an example of how to register a user and create a bookmark using `curl`:

1.  **Register a new user:**
    ```bash
    curl -X POST http://localhost:3000/api/auth/register \
    -H "Content-Type: application/json" \
    -d '{'
      "username": "testuser",
      "email": "test@example.com",
      "password": "password123"
    }'
    ```

2.  **Log in to get a token:**
    ```bash
    curl -X POST http://localhost:3000/api/auth/login \
    -H "Content-Type: application/json" \
    -d '{'
      "email": "test@example.com",
      "password": "password123"
    }'
    ```

3.  **Create a bookmark (replace `YOUR_TOKEN` with the token from the login response):**
    ```bash
    curl -X POST http://localhost:3000/api/bookmarks \
    -H "Authorization: Bearer YOUR_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{'
      "url": "https://golang.org/",
      "title": "The Go Programming Language",
      "tags": ["go", "programming"]
    }'
    ```

## Data Models

*   **User:** Represents a user with an ID, username, email, and password.
*   **Bookmark:** Represents a bookmark with a URL, title, description, notes, and associated tags.
*   **Tag:** Represents a tag with a name.

## Technologies Used

*   **Go:** The primary programming language.
*   **SQLite:** The database for storing data.
*   **JWT:** For user authentication.
