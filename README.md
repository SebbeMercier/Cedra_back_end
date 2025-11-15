# Cedra Back End

<!-- Replace this with the actual logo image -->
<!--
<p align="center">
  <img src="path/to/your/logo.png" alt="Cedra Logo" width="200"/>
</p>
-->

## Overview

The Cedra backend provides the core logic and data management for the Cedra application, an e-commerce or business management platform. Built with Go and the Gin web framework, it offers a RESTful API for user authentication, product and order management, and company administration. It integrates several key technologies: ScyllaDB for scalable data storage, Redis for caching and session management, Elasticsearch for product searching, MinIO for object storage, and Stripe for payment processing.

## Features

*   **User Authentication:**
    *   Local authentication with secure password storage using bcrypt.
    *   OAuth 2.0 support for Google and Facebook, managed with the Goth library.
    *   JWT (JSON Web Token) based authentication for API access.
*   **Product Management:**
    *   CRUD (Create, Read, Update, Delete) operations for products and categories.
    *   Image handling via MinIO, including upload, storage, and retrieval of product images.
    *   Full-text search functionality powered by Elasticsearch.
*   **Order Processing:**
    *   Manages the lifecycle of orders, from creation to fulfillment.
    *   Integration with Stripe for handling payment intents and processing payments.
    *   Automated invoice generation and delivery in PDF format.
*   **Company Administration:**
    *   Tools for managing company profiles, billing information, and employee accounts.
    *   Role-based access control (RBAC) to manage permissions within the application.
*   **Database Integration:**
    *   ScyllaDB: Primary data store, chosen for its high availability and scalability.
    *   Redis: Used for caching frequently accessed data, session management, and reset token storage.
    *   Elasticsearch: Powers the product search functionality, providing fast and relevant search results.
    *   MinIO: Stores product images and other static assets.

## Architecture

The backend application is structured into several distinct layers, each responsible for a specific aspect of the application's functionality:

*   **cmd/server:** Contains the `main.go` file, which serves as the entry point for the application. It's responsible for initializing the Gin router, registering routes, connecting to databases, and starting the HTTP server.
*   **internal/config:** Manages application configuration, loading settings from environment variables, and setting up OAuth providers.
*   **internal/database:** Establishes and manages connections to ScyllaDB, Redis, Elasticsearch, and MinIO. It provides methods for interacting with these databases.
*   **internal/handlers:** Implements the business logic for handling HTTP requests. Each handler corresponds to a specific API endpoint and performs tasks such as validating input, interacting with the database, and returning a response.
*   **internal/middleware:** Implements middleware functions for authentication, authorization, and other request processing tasks.
*   **internal/models:** Defines the data structures (structs) used throughout the application to represent entities such as users, products, orders, and addresses.
*   **internal/routes:** Defines the API endpoints and their corresponding handlers, using the Gin router.
*   **internal/services:** Implements reusable services such as indexing products in Elasticsearch and generating signed URLs for accessing MinIO objects.
*   **internal/utils:** Provides utility functions such as sending emails, generating invoice PDFs, and generating QR codes.

```mermaid
sequenceDiagram
    participant Client
    participant GinRouter
    participant AuthMiddleware
    participant Handler
    participant Database
    participant ExternalServices

    Client->>GinRouter: API Request (e.g., /api/products)
    GinRouter->>AuthMiddleware: Authentication Check (JWT)
    alt Authentication Failed
        AuthMiddleware->>Client: 401 Unauthorized
    else Authentication Successful
        AuthMiddleware->>Handler: Request
        Handler->>Database: Data Retrieval/Modification (ScyllaDB, Redis)
        Handler->>ExternalServices: (Elasticsearch, MinIO, Stripe)
        ExternalServices-->>Handler: Response
        Handler->>Client: Response
    end
