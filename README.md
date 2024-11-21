# Social Network Project

## Overview

This project is a Facebook-like social network application that includes features such as followers, profiles, posts, groups, notifications, and chats. The application is built with a focus on both frontend and backend development, utilizing modern web technologies and containerization for deployment.

## Features

- **Followers**: Users can follow and unfollow each other, with options for public and private profiles.
- **Profiles**: Each user has a profile displaying their information and activity.
- **Posts**: Users can create posts with privacy settings and comment on others' posts.
- **Groups**: Users can create and join groups, post within them, and create events.
- **Chats**: Private messaging and group chat functionality using Websockets.
- **Notifications**: Users receive notifications for various activities like follow requests and group invitations.

## Technologies Used

- **Frontend**: HTML, CSS, JavaScript, and a JS framework of your choice (e.g., Next.js, Vue.js, Svelte, Mithril).
- **Backend**: Go, SQLite for database management, and Caddy for the web server.
- **Containerization**: Docker for creating separate images for frontend and backend.
- **Authentication**: Sessions and cookies for user login and registration.
- **Websockets**: For real-time chat functionality.

## Setup Instructions

### Prerequisites

- Docker
- Go
- Next.js and npm (for frontend development)
- SQLite

### Installation

1. **Clone the Repository**

   ```bash
   git clone https://github.com/superrnovaa/Social_Network.git
   cd Social_Network
   ```

2. **Backend Setup**

   - Navigate to the backend directory:
     ```bash
     cd backend
     ```
   - Build the Docker image:
     ```bash
     docker build -t social-network-backend .
     ```
   - Run the Docker container:
     ```bash
     docker run -p 8080:8080 social-network-backend
     ```

3. **Frontend Setup**

   - Navigate to the frontend directory:
     ```bash
     cd frontend
     ```
   - Build the Docker image:
     ```bash
     docker build -t social-network-frontend .
     ```
   - Run the Docker container:
     ```bash
     docker run -p 3000:3000 social-network-frontend
     ```

4. **Database Migrations**

   - Ensure the SQLite database is set up and migrations are applied:
     ```bash
     go run backend/pkg/db/sqlite/sqlite.go
     ```

### Usage

- Access the frontend at `http://localhost:3000`.
- The backend API is available at `http://localhost:8080`.


## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

