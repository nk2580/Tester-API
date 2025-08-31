# Tester-API

## Description

The Tester-API is a simple Go-based REST API built using the Gin framework and GORM ORM, with SQLite as the database backend. It provides two main endpoints: a POST endpoint to register "pings" (messages) and a GET endpoint to retrieve all stored pings. This API serves as a validation tool for other systems.

## Prerequisites

- Go 1.20 or later
- Git (for cloning the repository)

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/nk2580/Tester-API.git
   ```

2. Navigate to the project directory:
   ```bash
   cd Tester-API
   ```

3. Install dependencies:
   ```bash
   go mod tidy
   ```

4. Run the application:
   ```bash
   go run main.go
   ```

The server will start on port 8080.

## Usage

Once the server is running, you can interact with the API using curl or any HTTP client.

### Register a Ping

Send a POST request to `/ping` with a JSON payload containing the message:

```bash
curl -X POST http://localhost:8080/ping \
  -H "Content-Type: application/json" \
  -d '{"message":"Hello, this is a test ping!"}'
```

Expected response:
```json
{"message": "Ping registered successfully!"}
```

### Retrieve All Pings

Send a GET request to `/pings` to retrieve all stored pings:

```bash
curl http://localhost:8080/pings
```

Expected response (example):
```json
[
  {"id": 1, "message": "Hello, this is a test ping!"}
]
```

## API Endpoints

### POST /ping

Registers a new ping message.

- **Method**: POST
- **Path**: `/ping`
- **Request Body**: JSON object with `message` field
  ```json
  {
    "message": "string"
  }
  ```
- **Success Response** (200):
  ```json
  {
    "message": "Ping registered successfully!"
  }
  ```
- **Error Responses**:
  - 400 Bad Request: If the request body is invalid
    ```json
    {
      "error": "error message"
    }
    ```
  - 500 Internal Server Error: If there is an issue saving to the database
    ```json
    {
      "error": "Failed to save ping"
    }
    ```

### GET /pings

Retrieves all stored ping messages.

- **Method**: GET
- **Path**: `/pings`
- **Request Body**: None
- **Success Response** (200): Array of ping objects
  ```json
  [
    {
      "id": 1,
      "message": "string"
    }
  ]
  ```
- **Error Response** (500):
  ```json
  {
    "error": "Failed to retrieve pings"
  }
  ```

## Contributing

We welcome contributions to the Tester-API project! To contribute:

1. Fork the repository on GitHub.
2. Create a new branch for your feature or bug fix:
   ```bash
   git checkout -b feature/your-feature-name
   ```
3. Make your changes and commit them:
   ```bash
   git commit -m "Add your commit message"
   ```
4. Push your changes to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```
5. Open a pull request on the main repository with a description of your changes.

Please ensure your code follows Go best practices and includes appropriate tests if applicable.

## License

This project is licensed under the GNU General Public License version 3 (GPL-3.0). See the [GPL-3.0 License](https://www.gnu.org/licenses/gpl-3.0.en.html) for details.
