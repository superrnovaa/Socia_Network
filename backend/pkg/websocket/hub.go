package websocket


type Hub struct {
    clients    map[int]*Client // Map user IDs to Client instances
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
}

func NewHub() *Hub {
    return &Hub{
        broadcast:  make(chan []byte),
        register:   make(chan *Client),
        unregister: make(chan *Client),
        clients:    make(map[int]*Client), // Updated to map user IDs to Client instances
    }
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.clients[client.userID] = client // Register the client using userID
        case client := <-h.unregister:
            if _, ok := h.clients[client.userID]; ok {
                delete(h.clients, client.userID) // Unregister the client using userID
                close(client.send) // Close the send channel
            }
        case message := <-h.broadcast:
            // Iterate over the clients map to send the message
            for userID, client := range h.clients {
                select {
                case client.send <- message: // Send the message to the client's send channel
                default:
                    close(client.send) // Close the send channel if not ready
                    delete(h.clients, userID) // Remove the client from the map
                }
            }
        }
    }
}
