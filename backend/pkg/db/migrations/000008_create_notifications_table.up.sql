CREATE TABLE notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    notifiedUser_id INTEGER NOT NULL,
    notifyingUser_id INTEGER NOT NULL,
    object TEXT NOT NULL,
    object_id INTEGER NOT NULL,
    type TEXT CHECK(type IN ('follow_request', 'group_invitation', 'group_join_request', 'event_creation', 'follow', 'post', 'reaction','comment','group')) NOT NULL,
    content TEXT NOT NULL,
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (notifiedUser_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (notifyingUser_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_notifications_type_object_id ON notifications(type, object_id);

-- Add a trigger to enforce referential integrity based on the notification type
CREATE TRIGGER check_notification_object_id
BEFORE INSERT ON notifications
FOR EACH ROW
BEGIN
    SELECT CASE
        WHEN NEW.type IN ('group_invitation', 'group_join_request', 'group') AND NOT EXISTS (SELECT 1 FROM groups WHERE id = NEW.object_id) THEN
            RAISE(ABORT, 'Invalid group_id for group-related notification')
        WHEN NEW.type IN ('post') AND NOT EXISTS (SELECT 1 FROM posts WHERE id = NEW.object_id) THEN
            RAISE(ABORT, 'Invalid post_id for post-related notification')
        WHEN NEW.type IN ('event_creation') AND NOT EXISTS (SELECT 1 FROM events WHERE id = NEW.object_id) THEN
            RAISE(ABORT, 'Invalid event_id for event-related notification')
        WHEN NEW.type IN ('follow', 'follow_request') AND NOT EXISTS (SELECT 1 FROM users WHERE id = NEW.object_id) THEN
            RAISE(ABORT, 'Invalid user_id for follow-related notification')
    END;
END;
