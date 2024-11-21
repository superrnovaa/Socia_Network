CREATE TABLE chat_notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id INTEGER NOT NULL,
    notifiedUser_id INTEGER NOT NULL,
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
    FOREIGN KEY (notifiedUser_id) REFERENCES users(id) ON DELETE CASCADE
);