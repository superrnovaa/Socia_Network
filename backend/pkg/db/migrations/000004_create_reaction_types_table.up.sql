CREATE TABLE reaction_types (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    icon_url TEXT NOT NULL
);

-- Insert default reactions with emojis
INSERT INTO reaction_types (name, icon_url) VALUES
('like', '👍'),
('dislike', '👎'),
('love', '❤️'),
('haha', '😂'),
('wow', '😮'),
('sad', '😢'),
('angry', '😡');