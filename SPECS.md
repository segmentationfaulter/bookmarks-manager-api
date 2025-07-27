# Bookmark Manager API - Project Specifications

## Project Overview

Build a RESTful API for managing bookmarks with user authentication using only Go's standard library.

## Core Features

- User registration and authentication with JWT tokens
- CRUD operations for bookmarks (URL, title, description, notes, tags)
- Tag management system
- Search and filter bookmarks by tags and text
- Pagination for bookmark listings
- SQLite database for data persistence

## Technical Requirements

- Use only Go standard library (no external frameworks)
- SQLite database with `database/sql` package
- JWT authentication
- JSON API responses
- RESTful endpoint design

## Data Models

### User
- ID, Username, Email, Password Hash, Created/Updated timestamps

### Bookmark
- ID, User ID, URL, Title, Description, Notes, Tags, Created/Updated timestamps

### Tag
- ID, User ID, Name, Created timestamp

## API Endpoints

### Authentication
- `POST /api/auth/register` - Register new user
- `POST /api/auth/login` - Login user
- `GET /api/auth/me` - Get current user profile

### Bookmarks
- `GET /api/bookmarks` - List bookmarks (with pagination, search, tag filtering)
- `POST /api/bookmarks` - Create bookmark
- `GET /api/bookmarks/{id}` - Get single bookmark
- `PUT /api/bookmarks/{id}` - Update bookmark
- `DELETE /api/bookmarks/{id}` - Delete bookmark

### Tags
- `GET /api/tags` - List user's tags
- `DELETE /api/tags/{id}` - Delete tag



## Request/Response Examples

### Register User
**POST** `/api/auth/register`
```json
{
  "username": "johndoe",
  "email": "john@example.com", 
  "password": "securepassword123"
}
```

### Login Response
```json
{
  "token": "jwt_token_here",
  "user": {
    "id": 1,
    "username": "johndoe",
    "email": "john@example.com"
  }
}
```

### Create Bookmark
**POST** `/api/bookmarks`
```json
{
  "url": "https://golang.org",
  "title": "Go Programming Language",
  "description": "Official Go website",
  "notes": "Great learning resource",
  "tags": ["programming", "go"]
}
```

### List Bookmarks Response
```json
{
  "bookmarks": [
    {
      "id": 1,
      "url": "https://golang.org",
      "title": "Go Programming Language", 
      "description": "Official Go website",
      "notes": "Great learning resource",
      "tags": ["programming", "go"],
      "created_at": "2025-01-08T12:00:00Z",
      "updated_at": "2025-01-08T12:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 1,
    "total_pages": 1
  }
}
```

## Query Parameters

### Bookmark Listing
- `page` - Page number (default: 1)
- `limit` - Items per page (default: 20, max: 100)
- `tags` - Comma-separated tag names for filtering
- `search` - Search in title, description, notes
- `sort` - Sort by: created_at, updated_at, title
- `order` - Sort order: asc, desc

## Authentication

- JWT tokens required for all bookmark and tag endpoints
- Include token in Authorization header: `Bearer <token>`
- Tokens expire in 24 hours

## Validation Rules

### User Registration
- Username: 3-50 characters, alphanumeric + underscore
- Email: Valid email format
- Password: Minimum 8 characters

### Bookmarks
- URL: Valid HTTP/HTTPS format
- Title: Maximum 500 characters
- Description: Maximum 2000 characters  
- Notes: Maximum 5000 characters
- Tags: Maximum 20 tags, each 1-50 characters

## Error Response Format

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable error message",
    "details": {}
  }
}
```

## HTTP Status Codes

- 200 OK - Successful GET/PUT
- 201 Created - Successful POST
- 204 No Content - Successful DELETE
- 400 Bad Request - Invalid input
- 401 Unauthorized - Missing/invalid auth
- 404 Not Found - Resource not found
- 409 Conflict - Duplicate resource
- 500 Internal Server Error - Server error

## Database Requirements

- SQLite database
- Users table with unique username/email constraints
- Bookmarks table with foreign key to users
- Tags table with user association
- Many-to-many relationship between bookmarks and tags
- Appropriate indexes for performance

## Success Criteria

- All endpoints functional with proper authentication
- Database operations work correctly
- Input validation prevents invalid data
- Pagination works for large datasets
- Search and filtering return accurate results
- Error handling provides meaningful responses
- Application can be containerized with Docker